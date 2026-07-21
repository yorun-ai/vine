package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

func TestCertificateUsesParsedLeafDomains(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"demo.local"})

	parsed, err := newCertificate(cert)

	require.NoError(t, err)
	assert.Equal(t, []string{"demo.local"}, parsed.domains)
}

func TestCertificateMatchesWildcardHost(t *testing.T) {
	cert := newTestPortalCert(t, "demo-cert", []string{"*.demo.local"})
	parsed, err := newCertificate(cert)
	require.NoError(t, err)

	assert.True(t, parsed.MatchesWildcardHost("admin.demo.local"))
	assert.False(t, parsed.MatchesWildcardHost("api.admin.demo.local"))
	assert.False(t, parsed.MatchesWildcardHost("demo.local"))
}

func newTestPortalCert(t *testing.T, name string, domains []string) *redised.PortalCert {
	t.Helper()

	certPEM, keyPEM := newTestCertificatePEM(t, domains)
	return &redised.PortalCert{
		Name:             name,
		Issuer:           "test",
		PublicKeyBase64:  base64.StdEncoding.EncodeToString(certPEM),
		PrivateKeyBase64: base64.StdEncoding.EncodeToString(keyPEM),
		ValidFrom:        time.Now().Add(-time.Hour),
		ValidTo:          time.Now().Add(time.Hour),
	}
}

func newTestCertificatePEM(t *testing.T, domains []string) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		DNSNames:              domains,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	return certPEM, keyPEM
}
