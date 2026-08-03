package vault

import (
	"context"
	"crypto/tls"
	"errors"
	"sort"
	"strings"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/mtls"
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/cacheutil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

var errCertificateNotFound = errors.New("entry certificate not found")

const maxMissingHostCacheSize = 1024

var vaultLogger = logger.New("daemon:portal:vault")

type Vault struct {
	app.BaseModule

	Redis    *hubredis.Client `inject:""`
	Context  context.Context  `inject:""`
	Identity *mtls.Identity   `inject:""`

	mutex      sync.RWMutex
	certs      map[string]*_Certificate
	namesByKey map[string]string

	certsByHost   map[string]*_Certificate
	missingHosts  *cacheutil.LruSet[string]
	wildcardCerts []*_Certificate

	temporaryWebCerts *_TemporaryWebCertificates
}

func (v *Vault) DIInit() {
	v.initTemporaryWebCerts()
	v.certs = map[string]*_Certificate{}
	v.namesByKey = map[string]string{}
	v.rebuildIndexLocked()
	valuesByKey, subscription := v.Redis.LoadListAndSubscribe(v.Context, redised.FormatPortalCertPrefix(), v.handleCertEvent)
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
	host := strings.ToLower(strings.TrimSuffix(hello.ServerName, "."))

	v.mutex.RLock()
	cert := v.certsByHost[host]
	temporaryWebCerts := v.temporaryWebCerts
	missing := temporaryWebCerts == nil && host != "" && v.missingHosts.Contains(host)
	v.mutex.RUnlock()
	if cert != nil {
		return cert.cert, nil
	}
	if host == "" {
		if temporaryWebCerts != nil {
			return temporaryWebCerts.Certificate("")
		}
		return nil, errCertificateNotFound
	}
	if missing {
		return nil, errCertificateNotFound
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()

	cert = v.matchWildcardCertLocked(host)
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

func (v *Vault) loadCerts(valuesByKey map[string]string, subscription hubapiredis.Subscription) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for key, value := range valuesByKey {
		cert := vcode.MustUnmarshalJsonS[*redised.PortalCert](value)
		v.setCertLocked(cert)
		v.namesByKey[key] = cert.Name
	}
	v.rebuildIndexLocked()
	subscription.Start()
}

func (v *Vault) handleCertEvent(event hubapiredis.Event) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	name := v.namesByKey[event.Key]
	if event.Kind == hubapiredis.EventKindDelete {
		delete(v.certs, name)
		delete(v.namesByKey, event.Key)
		v.rebuildIndexLocked()
		return
	}

	cert := vcode.MustUnmarshalJsonS[*redised.PortalCert](event.Value)
	delete(v.certs, name)
	v.setCertLocked(cert)
	v.namesByKey[event.Key] = cert.Name
	v.rebuildIndexLocked()
}

func (v *Vault) setCertLocked(cert *redised.PortalCert) {
	parsed, err := newCertificate(cert)
	if err != nil {
		vaultLogger.Error("vine.portal entry cert ignored", "name", cert.Name, "error", err)
		delete(v.certs, cert.Name)
		return
	}
	v.certs[cert.Name] = parsed
}

func (v *Vault) rebuildIndexLocked() {
	v.certsByHost = map[string]*_Certificate{}
	v.missingHosts = cacheutil.NewLruSet[string](maxMissingHostCacheSize)
	v.wildcardCerts = nil

	names := make([]string, 0, len(v.certs))
	for name := range v.certs {
		names = append(names, name)
	}
	sort.Strings(names)

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
}

func (v *Vault) matchWildcardCertLocked(host string) *_Certificate {
	for _, cert := range v.wildcardCerts {
		if cert.MatchesWildcardHost(host) {
			return cert
		}
	}
	return nil
}
