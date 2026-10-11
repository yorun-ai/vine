package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortalCertCoreCreateDerivesMetadata(t *testing.T) {
	repo := newTestPortalCertRepo()
	core := &PortalCertCore{PortalCertRepo: repo, PortalEntryRepo: newPortalEntryRepoSpy()}
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	certificate, privateKey := testPortalCertPEM(t, "letsencrypt", []string{"demo.local", "*.demo.local"}, []net.IP{net.ParseIP("127.0.0.1")}, validFrom, validTo)
	cert := core.Create(PortalCertCreation{
		Name:        "demo-cert",
		Certificate: certificate,
		PrivateKey:  privateKey,
	})

	assert.Equal(t, "letsencrypt", cert.Issuer)
	assert.Equal(t, []string{"demo.local", "*.demo.local", "127.0.0.1"}, cert.Domains)
	assert.True(t, cert.ValidFrom.Equal(validFrom))
	assert.True(t, cert.ValidTo.Equal(validTo))
}

func TestPortalCertCoreUpdateDerivesMetadataWhenPublicKeyChanges(t *testing.T) {
	repo := newTestPortalCertRepo()
	core := &PortalCertCore{PortalCertRepo: repo, PortalEntryRepo: newPortalEntryRepoSpy()}
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	certificate, privateKey := testPortalCertPEM(t, "letsencrypt", []string{"demo.local"}, nil, validFrom, validTo)
	cert := core.Create(PortalCertCreation{
		Name:        "demo-cert",
		Certificate: certificate,
		PrivateKey:  privateKey,
	})
	nextFrom := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	nextTo := time.Date(2028, 2, 1, 0, 0, 0, 0, time.UTC)
	nextPublicKey, nextPrivateKey := testPortalCertPEM(t, "next-ca", []string{"next.local"}, nil, nextFrom, nextTo)

	got := core.Update(cert.Id, PortalCertUpdate{
		Certificate: &nextPublicKey,
		PrivateKey:  &nextPrivateKey,
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

func (r *_TestPortalCertRepo) List() []*PortalCert {
	ret := make([]*PortalCert, 0, len(r.certs))
	for _, cert := range r.certs {
		ret = append(ret, cert)
	}
	return ret
}

func (r *_TestPortalCertRepo) GetById(id int) (*PortalCert, bool) {
	cert, ok := r.certs[id]
	return cert, ok
}

func (r *_TestPortalCertRepo) GetByName(name string) (*PortalCert, bool) {
	id, ok := r.names[name]
	if !ok {
		return nil, false
	}
	return r.certs[id], true
}

func (r *_TestPortalCertRepo) Save(cert *PortalCert) {
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

func (r *_TestPortalCertRepo) Remove(id int) bool {
	cert, ok := r.certs[id]
	if !ok {
		return false
	}
	delete(r.certs, id)
	delete(r.names, cert.Name)
	return true
}

func testPortalCertPEM(t *testing.T, issuer string, dnsNames []string, ipAddresses []net.IP, validFrom time.Time, validTo time.Time) (string, string) {
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
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

func TestPortalCertSaveDerivesMetadataAndPreservesIdentity(t *testing.T) {
	repo := newTestPortalCertRepo()
	target := &PortalCertCore{PortalCertRepo: repo, PortalEntryRepo: newPortalEntryRepoSpy()}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(1, 0, 0)
	certificate, privateKey := testPortalCertPEM(t, "actual", []string{"demo.local"}, nil, from, to)
	cert := PortalCert{Id: 99, Name: "demo", Enabled: true, Issuer: "forged", Domains: []string{"forged.local"}, Certificate: certificate, PrivateKey: privateKey}
	got := target.Save(cert)
	require.NotEqual(t, 99, got.Id)
	require.Equal(t, "actual", got.Issuer)
	require.Equal(t, []string{"demo.local"}, got.Domains)
	require.Equal(t, from, got.ValidFrom)
	require.Equal(t, to, got.ValidTo)
	require.Equal(t, got.Id, target.Save(cert).Id)
	require.Len(t, repo.certs, 1)
	cert.Certificate = "invalid"
	require.Panics(t, func() { target.Save(cert) })
	require.Equal(t, got.Certificate, repo.certs[got.Id].Certificate)
	require.NotPanics(t, func() { (&PortalCertCore{}).Validate(*got) })
}

func TestPortalCertPEMValidationAndUpdates(t *testing.T) {
	from := time.Now().Add(-time.Hour)
	certificate, key := testPortalCertPEM(t, "issuer", []string{"demo.local"}, nil, from, from.Add(2*time.Hour))
	_, otherKey := testPortalCertPEM(t, "other", []string{"other.local"}, nil, from, from.Add(2*time.Hour))
	target := &PortalCertCore{PortalCertRepo: newTestPortalCertRepo(), PortalEntryRepo: newPortalEntryRepoSpy()}
	chain := certificate + certificate
	stored := target.Create(PortalCertCreation{Name: "chain", Certificate: chain, PrivateKey: key})
	require.Equal(t, chain, stored.Certificate)
	updated := target.Update(stored.Id, PortalCertUpdate{Certificate: &certificate})
	require.Equal(t, key, updated.PrivateKey, "omitted key keeps the stored key")
	for _, change := range []PortalCertUpdate{
		{PrivateKey: &otherKey}, {PrivateKey: new("")},
		{Certificate: new(base64.StdEncoding.EncodeToString([]byte(certificate)))},
		{Certificate: new(certificate + string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("bad DER")})))},
	} {
		require.Panics(t, func() { target.Update(stored.Id, change) })
		require.Equal(t, key, target.Get(stored.Id).PrivateKey)
		require.Equal(t, certificate, target.Get(stored.Id).Certificate)
	}
}

func TestPortalCertDisabledPlaceholderRequiresValidPEMOnEnable(t *testing.T) {
	target := &PortalCertCore{PortalCertRepo: newTestPortalCertRepo(), PortalEntryRepo: newPortalEntryRepoSpy()}
	require.Panics(t, func() { target.Create(PortalCertCreation{Enabled: new(false)}) })
	cert := target.Create(PortalCertCreation{Name: "placeholder", Enabled: new(false)})
	require.Empty(t, cert.Certificate)
	require.Empty(t, cert.PrivateKey)
	cert = target.Update(cert.Id, PortalCertUpdate{Certificate: new("pending"), PrivateKey: new("pending")})
	require.False(t, cert.Enabled)
	require.Equal(t, "pending", cert.Certificate)
	require.Panics(t, func() { target.Update(cert.Id, PortalCertUpdate{Enabled: new(true)}) })
	require.False(t, target.Get(cert.Id).Enabled, "a failed enable leaves the stored certificate disabled")

	from := time.Now().Add(-time.Hour)
	certificate, key := testPortalCertPEM(t, "issuer", []string{"demo.local"}, nil, from, from.Add(2*time.Hour))
	cert = target.Update(cert.Id, PortalCertUpdate{Certificate: &certificate, PrivateKey: &key, Enabled: new(true)})
	require.True(t, cert.Enabled)
	require.Equal(t, []string{"demo.local"}, cert.Domains)
}

func TestPortalCertChangesPreserveExplicitEntryRequirements(t *testing.T) {
	from := time.Now().Add(-2 * time.Hour)
	certificate, key := testPortalCertPEM(t, "issuer", []string{"*.example.com"}, nil, from, from.Add(time.Hour))
	otherCertificate, otherKey := testPortalCertPEM(t, "issuer", []string{"other.local"}, nil, from, from.Add(time.Hour))
	for _, redirect := range []bool{false, true} {
		repo := newTestPortalCertRepo()
		entries := newPortalEntryRepoSpy()
		target := new(PortalCertCore{PortalCertRepo: repo, PortalEntryRepo: entries})
		cert := target.Create(PortalCertCreation{Name: "primary", Certificate: certificate, PrivateKey: key})
		raw := PortalEntryHTTP{Host: "app.example.com"}
		if redirect {
			raw.AutoHTTPS = PortalSwitchModeEnabled
		} else {
			raw.Https = PortalSwitchModeEnabled
		}
		entry := new(PortalEntry{Name: "web", Protocol: "http", Enabled: true, Http: &raw})
		entries.Save(entry)
		for _, change := range []func(){
			func() { target.Remove(cert.Id) },
			func() { target.Update(cert.Id, PortalCertUpdate{Enabled: new(false)}) },
			func() {
				target.Update(cert.Id, PortalCertUpdate{Certificate: &otherCertificate, PrivateKey: &otherKey})
			},
			func() {
				target.Save(PortalCert{Name: cert.Name, Enabled: true, Certificate: otherCertificate, PrivateKey: otherKey})
			},
		} {
			require.Panics(t, change)
			require.Equal(t, cert, target.Get(cert.Id), "a rejected change must leave the certificate intact")
		}
		// A second matching certificate permits removal of the first.
		backup := target.Create(PortalCertCreation{Name: "backup", Certificate: certificate, PrivateKey: key})
		require.NotPanics(t, func() { target.Remove(cert.Id) })
		require.Panics(t, func() { target.Remove(backup.Id) })
		// Automatic entries allow the last matching certificate to be removed.
		entry.Http = new(PortalEntryHTTP{Host: "app.example.com"})
		entries.Save(entry)
		require.NotPanics(t, func() { target.Remove(backup.Id) })
	}
}
