package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortalCertCoreCreateDerivesMetadata(t *testing.T) {
	repo := newTestPortalCertRepo()
	core := &PortalCertCore{PortalCertRepo: repo}
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	cert := core.Create(PortalCertCreation{
		Name:             "demo-cert",
		PublicKeyBase64:  testPortalCertBase64(t, "letsencrypt", []string{"demo.local", "*.demo.local"}, []net.IP{net.ParseIP("127.0.0.1")}, validFrom, validTo),
		PrivateKeyBase64: "pri",
	})

	assert.Equal(t, "letsencrypt", cert.Issuer)
	assert.Equal(t, []string{"demo.local", "*.demo.local", "127.0.0.1"}, cert.Domains)
	assert.True(t, cert.ValidFrom.Equal(validFrom))
	assert.True(t, cert.ValidTo.Equal(validTo))
}

func TestPortalCertCoreUpdateDerivesMetadataWhenPublicKeyChanges(t *testing.T) {
	repo := newTestPortalCertRepo()
	core := &PortalCertCore{PortalCertRepo: repo}
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	cert := core.Create(PortalCertCreation{
		Name:             "demo-cert",
		PublicKeyBase64:  testPortalCertBase64(t, "letsencrypt", []string{"demo.local"}, nil, validFrom, validTo),
		PrivateKeyBase64: "pri",
	})
	nextFrom := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	nextTo := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)
	nextPublicKey := testPortalCertBase64(t, "next-ca", []string{"next.local"}, nil, nextFrom, nextTo)

	got := core.Update(cert.Id, PortalCertUpdate{
		PublicKeyBase64: &nextPublicKey,
	})

	assert.Equal(t, "next-ca", got.Issuer)
	assert.Equal(t, []string{"next.local"}, got.Domains)
	assert.True(t, got.ValidFrom.Equal(nextFrom))
	assert.True(t, got.ValidTo.Equal(nextTo))
}

type _TestPortalCertRepo struct {
	nextId int
	certs  map[int]*PortalCert
	names  map[string]int
}

func newTestPortalCertRepo() *_TestPortalCertRepo {
	return &_TestPortalCertRepo{
		nextId: 1,
		certs:  map[int]*PortalCert{},
		names:  map[string]int{},
	}
}

func (r *_TestPortalCertRepo) ListCerts() []*PortalCert {
	ret := make([]*PortalCert, 0, len(r.certs))
	for _, cert := range r.certs {
		ret = append(ret, cert)
	}
	return ret
}

func (r *_TestPortalCertRepo) GetCertById(id int) (*PortalCert, bool) {
	cert, ok := r.certs[id]
	return cert, ok
}

func (r *_TestPortalCertRepo) GetCertByName(name string) (*PortalCert, bool) {
	id, ok := r.names[name]
	if !ok {
		return nil, false
	}
	return r.certs[id], true
}

func (r *_TestPortalCertRepo) SaveCert(cert *PortalCert) {
	if cert.Id == 0 {
		cert.Id = r.nextId
		r.nextId++
	}
	for name, id := range r.names {
		if id == cert.Id && name != cert.Name {
			delete(r.names, name)
		}
	}
	r.certs[cert.Id] = cert
	r.names[cert.Name] = cert.Id
}

func (r *_TestPortalCertRepo) RemoveCert(id int) bool {
	cert, ok := r.certs[id]
	if !ok {
		return false
	}
	delete(r.certs, id)
	delete(r.names, cert.Name)
	return true
}

func testPortalCertBase64(t *testing.T, issuer string, dnsNames []string, ipAddresses []net.IP, validFrom time.Time, validTo time.Time) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(validFrom.UnixNano()),
		Subject: pkix.Name{
			CommonName: "demo.local",
		},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		NotBefore:   validFrom,
		NotAfter:    validTo,
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	parent := &x509.Certificate{
		SerialNumber: big.NewInt(validFrom.UnixNano() + 1),
		Subject: pkix.Name{
			CommonName: issuer,
		},
		NotBefore: validFrom,
		NotAfter:  validTo,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
}
