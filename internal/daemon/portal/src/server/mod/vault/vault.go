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
	hubapiredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/cacheutil"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/util/vcode"
)

var errCertificateNotFound = errors.New("entry certificate not found")

const maxMissingHostCacheSize = 1024

type Vault struct {
	app.BaseModule

	Redis   *hubredis.Client `inject:""`
	Context context.Context  `inject:""`

	mutex      sync.RWMutex
	certs      map[string]*_Certificate
	namesByKey map[string]string

	certsByHost   map[string]*_Certificate
	missingHosts  *cacheutil.LruSet[string]
	wildcardCerts []*_Certificate
}

func (v *Vault) DIInit() {
	v.certs = map[string]*_Certificate{}
	v.namesByKey = map[string]string{}
	v.rebuildIndexLocked()
	valuesByKey := v.Redis.LoadListAndSubscribe(v.Context, redised.FormatPortalCertPrefix(), v.handleCertEvent)
	v.loadCerts(valuesByKey)
}

func (v *Vault) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := strings.ToLower(hello.ServerName)
	if host == "" {
		return nil, errCertificateNotFound
	}

	v.mutex.RLock()
	cert := v.certsByHost[host]
	missing := v.missingHosts.Contains(host)
	v.mutex.RUnlock()
	if cert != nil {
		return cert.cert, nil
	}
	if missing {
		return nil, errCertificateNotFound
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()

	cert = v.matchWildcardCertLocked(host)
	if cert == nil {
		v.missingHosts.Add(host)
		return nil, errCertificateNotFound
	}
	v.certsByHost[host] = cert
	return cert.cert, nil
}

func (v *Vault) loadCerts(valuesByKey map[string]string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for key, value := range valuesByKey {
		cert := vcode.MustUnmarshalJsonS[*redised.PortalCert](value)
		v.setCertLocked(cert)
		v.namesByKey[key] = cert.Name
	}
	v.rebuildIndexLocked()
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
		logger.Error("vine.portal entry cert ignored", "name", cert.Name, "error", err)
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
