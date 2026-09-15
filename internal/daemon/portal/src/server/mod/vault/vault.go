package vault

import (
	"context"
	"crypto/tls"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/cacheutil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubwatch"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

var errCertificateNotFound = errors.New("entry certificate not found")

const maxMissingHostCacheSize = 1024

var vaultLogger = logger.New("daemon:portal:vault")

type Vault struct {
	app.BaseModule

	Watch    *hubwatch.Client `inject:""`
	Context  context.Context  `inject:""`
	Identity *mtls.Identity   `inject:""`

	mutex      sync.RWMutex
	certs      map[string]*_Certificate
	namesByKey map[string]string

	certsByHost    map[string]*_Certificate
	missingHosts   *cacheutil.LruSet[string]
	wildcardCerts  []*_Certificate
	indexBuiltAt   time.Time
	indexExpiresAt time.Time

	temporaryWebCerts *_TemporaryWebCertificates
}

func (v *Vault) DIInit() {
	v.initTemporaryWebCerts()
	v.certs = map[string]*_Certificate{}
	v.namesByKey = map[string]string{}
	v.rebuildIndexLocked()
	valuesByKey, subscription := v.Watch.LoadListAndSubscribe(v.Context, watched.FormatPortalCertPrefix(), v.handleCertEvent)
	v.loadCerts(valuesByKey, subscription)
}

func (v *Vault) initTemporaryWebCerts() {
	if v.Identity.Enabled() {
		var err error
		v.temporaryWebCerts, err = newTemporaryWebCertificates()
		vpre.CheckNilError(err, "create Portal temporary Web certificate signer failed")
	}
}

func (v *Vault) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return v.getCertificateAt(hello, time.Now())
}

func (v *Vault) getCertificateAt(hello *tls.ClientHelloInfo, now time.Time) (*tls.Certificate, error) {
	host := strings.ToLower(strings.TrimSuffix(hello.ServerName, "."))

	v.mutex.RLock()
	fresh := v.indexFreshLocked(now)
	cert := v.certsByHost[host]
	temporaryWebCerts := v.temporaryWebCerts
	missing := temporaryWebCerts == nil && host != "" && v.missingHosts.Contains(host)
	v.mutex.RUnlock()
	if fresh && cert != nil {
		return cert.cert, nil
	}
	if host == "" {
		if temporaryWebCerts != nil {
			return temporaryWebCerts.Certificate("")
		}
		return nil, errCertificateNotFound
	}
	if fresh && missing {
		return nil, errCertificateNotFound
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()

	if !v.indexFreshLocked(now) {
		v.rebuildIndexAtLocked(now)
	}
	// Recheck the exact/cache entry after upgrading the lock.
	cert = v.certsByHost[host]
	if cert == nil {
		cert = v.matchWildcardCertLocked(host)
	}
	if cert != nil {
		v.certsByHost[host] = cert
		return cert.cert, nil
	}
	if temporaryWebCerts == nil {
		v.missingHosts.Add(host)
		return nil, errCertificateNotFound
	}
	return temporaryWebCerts.Certificate(host)
}

func (v *Vault) loadCerts(valuesByKey map[string]string, subscription hubapiwatch.Subscription) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for key, value := range valuesByKey {
		cert := vcode.MustUnmarshalJsonS[*watched.PortalCert](value)
		v.setCertLocked(cert)
		v.namesByKey[key] = cert.Name
	}
	v.rebuildIndexLocked()
	subscription.Start()
}

func (v *Vault) handleCertEvent(event hubapiwatch.Event) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	name := v.namesByKey[event.Key]
	if event.Kind == hubapiwatch.EventKindDelete {
		delete(v.certs, name)
		delete(v.namesByKey, event.Key)
		v.rebuildIndexLocked()
		return
	}

	cert := vcode.MustUnmarshalJsonS[*watched.PortalCert](event.Value)
	delete(v.certs, name)
	v.setCertLocked(cert)
	v.namesByKey[event.Key] = cert.Name
	v.rebuildIndexLocked()
}

func (v *Vault) setCertLocked(cert *watched.PortalCert) {
	parsed, err := newCertificate(cert)
	if err != nil {
		vaultLogger.Error("vine.portal entry cert ignored", "name", cert.Name, "error", err)
		delete(v.certs, cert.Name)
		return
	}
	v.certs[cert.Name] = parsed
}

func (v *Vault) indexFreshLocked(now time.Time) bool {
	return !now.Before(v.indexBuiltAt) && (v.indexExpiresAt.IsZero() || now.Before(v.indexExpiresAt))
}

func (v *Vault) rebuildIndexLocked() {
	v.rebuildIndexAtLocked(time.Now())
}

func (v *Vault) rebuildIndexAtLocked(now time.Time) {
	// Certificate validity follows wall time, including clock corrections.
	v.indexBuiltAt = now.Round(0)
	v.indexExpiresAt = time.Time{}
	v.certsByHost = map[string]*_Certificate{}
	v.missingHosts = cacheutil.NewLruSet[string](maxMissingHostCacheSize)
	v.wildcardCerts = nil

	names := make([]string, 0, len(v.certs))
	for name := range v.certs {
		names = append(names, name)
		cert := v.certs[name]
		// X.509 validity includes both endpoints; expiry starts just after validTo.
		for _, boundary := range []time.Time{cert.validFrom, cert.validTo.Add(time.Nanosecond)} {
			if boundary.After(now) && (v.indexExpiresAt.IsZero() || boundary.Before(v.indexExpiresAt)) {
				v.indexExpiresAt = boundary
			}
		}
	}
	sort.Slice(names, func(i int, j int) bool {
		a, b := v.certs[names[i]], v.certs[names[j]]
		if a.validAt(now) != b.validAt(now) {
			return a.validAt(now)
		}
		return names[i] < names[j]
	})

	for _, name := range names {
		cert := v.certs[name]
		hasWildcard := false
		for _, domain := range cert.domains {
			if strings.HasPrefix(domain, "*.") {
				hasWildcard = true
				continue
			}
			if _, ok := v.certsByHost[domain]; !ok {
				v.certsByHost[domain] = cert
			}
		}
		if hasWildcard {
			v.wildcardCerts = append(v.wildcardCerts, cert)
		}
	}
	// A valid wildcard outranks an exact match outside its validity period.
	for host, cert := range v.certsByHost {
		if !cert.validAt(now) {
			wildcard := v.matchWildcardCertLocked(host)
			if wildcard != nil && wildcard.validAt(now) {
				v.certsByHost[host] = wildcard
			}
		}
	}
}

func (v *Vault) matchWildcardCertLocked(host string) *_Certificate {
	for _, cert := range v.wildcardCerts {
		if cert.MatchesWildcardHost(host) {
			return cert
		}
	}
	return nil
}
