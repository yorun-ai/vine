package vault

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
