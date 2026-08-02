package vault

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
	"go.yorun.ai/vine/internal/testutil/mtlstest"
)

func TestVaultGetCertificateMatchesHost(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"demo.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := &Vault{
		certs: map[string]*_Certificate{
			parsed.name: parsed,
		},
	}
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "demo.local"})

	require.NoError(t, err)
	assert.Same(t, parsed.cert, got)
}

func TestVaultGetCertificateMatchesWildcardHost(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"*.demo.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := &Vault{
		certs: map[string]*_Certificate{
			parsed.name: parsed,
		},
	}
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "admin.demo.local"})

	require.NoError(t, err)
	assert.Same(t, parsed.cert, got)
	assert.Same(t, parsed, vault.certsByHost["admin.demo.local"])
}

func TestVaultGetCertificateDoesNotMatchMissingHost(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"demo.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := &Vault{
		certs: map[string]*_Certificate{
			parsed.name: parsed,
		},
	}
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "other.local"})

	assert.ErrorIs(t, err, errCertificateNotFound)
	assert.Nil(t, got)
	assert.True(t, vault.missingHosts.Contains("other.local"))
}

func TestVaultGetCertificateReturnsExpiredMatchingCert(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"demo.local"})
	cert.ValidFrom = time.Now().Add(-2 * time.Hour)
	cert.ValidTo = time.Now().Add(-time.Hour)
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := &Vault{
		certs: map[string]*_Certificate{
			parsed.name: parsed,
		},
	}
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "demo.local"})

	require.NoError(t, err)
	assert.Same(t, parsed.cert, got)
}

func TestVaultGetCertificatePrefersExactHostOverWildcard(t *testing.T) {
	wildcardCert := newTestPortalCert(t, "wildcard-cert", []string{"*.demo.local"})
	exactCert := newTestPortalCert(t, "exact-cert", []string{"admin.demo.local"})
	parsedWildcard, err := newCertificate(wildcardCert)
	require.NoError(t, err)
	parsedExact, err := newCertificate(exactCert)
	require.NoError(t, err)
	vault := &Vault{
		certs: map[string]*_Certificate{
			parsedWildcard.name: parsedWildcard,
			parsedExact.name:    parsedExact,
		},
	}
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "admin.demo.local"})

	require.NoError(t, err)
	assert.Same(t, parsedExact.cert, got)
}

func TestVaultGetCertificateUsesTemporaryWebCertWithMTLS(t *testing.T) {
	vault := newTemporaryWebCertVault(t, nil)

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})

	require.NoError(t, err)
	require.NotNil(t, got.Leaf)
	assert.NoError(t, got.Leaf.VerifyHostname("dashboard.local"))
	assert.Equal(t, []string{"Vine Portal"}, got.Leaf.Subject.Organization)
	assert.Equal(t, got.Leaf.RawSubject, got.Leaf.RawIssuer)
	assert.Empty(t, got.Leaf.URIs)
}

func TestVaultGetCertificateCachesTemporaryWebCert(t *testing.T) {
	vault := newTemporaryWebCertVault(t, nil)

	first, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})
	require.NoError(t, err)
	second, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})
	require.NoError(t, err)

	assert.Same(t, first, second)
}

func TestVaultGetCertificateTemporaryWebCertSupportsMissingSNI(t *testing.T) {
	vault := newTemporaryWebCertVault(t, nil)

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{})

	require.NoError(t, err)
	assert.NoError(t, got.Leaf.VerifyHostname("localhost"))
	assert.NoError(t, got.Leaf.VerifyHostname("127.0.0.1"))
	assert.NoError(t, got.Leaf.VerifyHostname("::1"))
}

func TestVaultGetCertificatePrefersConfiguredCertOverTemporaryWebCert(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"dashboard.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := newTemporaryWebCertVault(t, map[string]*_Certificate{parsed.name: parsed})

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})

	require.NoError(t, err)
	assert.Same(t, parsed.cert, got)
}

func TestVaultGetCertificateReplacesTemporaryWebCertWithConfiguredCert(t *testing.T) {
	vault := newTemporaryWebCertVault(t, nil)
	temporary, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})
	require.NoError(t, err)
	cert := newTestPortalCert(t, "dashboard-cert", []string{"dashboard.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault.mutex.Lock()
	vault.certs[parsed.name] = parsed
	vault.rebuildIndexLocked()
	vault.mutex.Unlock()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})

	require.NoError(t, err)
	assert.NotSame(t, temporary, got)
	assert.Same(t, parsed.cert, got)
}

func TestVaultGetCertificatePrefersConfiguredWildcardOverTemporaryWebCert(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"*.dashboard.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)
	vault := newTemporaryWebCertVault(t, map[string]*_Certificate{parsed.name: parsed})

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "admin.dashboard.local"})

	require.NoError(t, err)
	assert.Same(t, parsed.cert, got)
}

func TestVaultGetCertificateRejectsInvalidTemporaryWebCertHost(t *testing.T) {
	vault := newTemporaryWebCertVault(t, nil)

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "invalid_host.local"})

	assert.ErrorIs(t, err, errInvalidTemporaryWebCertificateHost)
	assert.Nil(t, got)
}

func TestVaultDoesNotEnableTemporaryWebCertWithoutMTLS(t *testing.T) {
	vault := &Vault{
		Identity: mtls.DisabledIdentity(),
		certs:    map[string]*_Certificate{},
	}
	vault.initTemporaryWebCerts()
	vault.rebuildIndexLocked()

	got, err := vault.GetCertificate(&tls.ClientHelloInfo{ServerName: "dashboard.local"})

	assert.ErrorIs(t, err, errCertificateNotFound)
	assert.Nil(t, got)
	assert.Nil(t, vault.temporaryWebCerts)
}

func newTemporaryWebCertVault(t *testing.T, certs map[string]*_Certificate) *Vault {
	t.Helper()
	if certs == nil {
		certs = map[string]*_Certificate{}
	}
	vault := &Vault{
		Identity: mtlstest.NewCA(t).Identity(t, mtls.PortalIdentity),
		certs:    certs,
	}
	vault.initTemporaryWebCerts()
	vault.rebuildIndexLocked()
	return vault
}
