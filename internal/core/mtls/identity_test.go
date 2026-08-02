package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTrustDomain    = "test.vine.local"
	testServerIdentity = SPIFFEPath("/test/server")
	testClientIdentity = SPIFFEPath("/test/client")
	testOtherIdentity  = SPIFFEPath("/test/other")
)

type testCA struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
	pem  []byte
}

func newTestCA(t *testing.T) testCA {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Vine test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return testCA{cert: cert, key: key, pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}

func (ca testCA) files(t *testing.T, identityPath SPIFFEPath, usages ...x509.ExtKeyUsage) Files {
	t.Helper()
	return ca.filesWithURIs(t, identityPath.String(), []string{"spiffe://" + testTrustDomain + identityPath.String()}, usages...)
}

func (ca testCA) filesWithURIs(t *testing.T, commonName string, uriStrings []string, usages ...x509.ExtKeyUsage) Files {
	t.Helper()
	if len(usages) == 0 {
		usages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}
	spiffeURIs := make([]*url.URL, 0, len(uriStrings))
	for _, uriString := range uriStrings {
		uri, err := url.Parse(uriString)
		require.NoError(t, err)
		spiffeURIs = append(spiffeURIs, uri)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		URIs:         spiffeURIs,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  usages,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)

	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(caFile, ca.pem, 0o600))
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600))
	return Files{CAFile: caFile, CertFile: certFile, KeyFile: keyFile}
}

func TestLoadDisabledIdentity(t *testing.T) {
	identity, err := Load(testServerIdentity, Files{})
	require.NoError(t, err)
	assert.False(t, identity.Enabled())
}

func TestLoadRequiresAllFiles(t *testing.T) {
	_, err := Load(testServerIdentity, Files{CAFile: "ca.pem"})
	assert.EqualError(t, err, "mtls-ca-file, mtls-cert-file, and mtls-key-file must be configured together")
}

func TestLoadValidatesIdentityAndBothUsages(t *testing.T) {
	ca := newTestCA(t)

	_, err := Load(testServerIdentity, ca.files(t, testClientIdentity))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match expected ID")

	_, err = Load(testServerIdentity, ca.filesWithURIs(t, testServerIdentity.String(), []string{
		"spiffe://" + testTrustDomain + testServerIdentity.String(),
		"spiffe://" + testTrustDomain + testClientIdentity.String(),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more than one URI SAN")

	_, err = Load(testServerIdentity, ca.filesWithURIs(t, testServerIdentity.String(), nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no URI SAN")

	_, err = Load(testServerIdentity, ca.files(t, testServerIdentity, x509.ExtKeyUsageServerAuth))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client authentication")
}

func TestLoadExposesExactSPIFFEID(t *testing.T) {
	ca := newTestCA(t)
	server, err := Load(testServerIdentity, ca.files(t, testServerIdentity))
	require.NoError(t, err)

	assert.Equal(t, "spiffe://"+testTrustDomain+testServerIdentity.String(), server.SPIFFEID())
	assert.Empty(t, DisabledIdentity().SPIFFEID())
}

func TestIdentityRejectsPeerFromDifferentTrustDomain(t *testing.T) {
	ca := newTestCA(t)
	server, err := Load(testServerIdentity, ca.files(t, testServerIdentity))
	require.NoError(t, err)
	client, err := Load(testClientIdentity, ca.filesWithURIs(t, testClientIdentity.String(), []string{
		"spiffe://other.vine.local" + testClientIdentity.String(),
	}))
	require.NoError(t, err)

	_, err = server.verifyPeer([]*x509.Certificate{client.certificate.Leaf}, x509.ExtKeyUsageClientAuth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no X.509 bundle found for trust domain")
}

func TestIdentityMutualTLSHandshake(t *testing.T) {
	ca := newTestCA(t)
	server, err := Load(testServerIdentity, ca.files(t, testServerIdentity))
	require.NoError(t, err)
	client, err := Load(testClientIdentity, ca.files(t, testClientIdentity))
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	tlsListener := tls.NewListener(listener, server.ServerConfig(testClientIdentity))
	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := tlsListener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer conn.Close()
		serverErr <- conn.(*tls.Conn).Handshake()
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), client.ClientConfig(testServerIdentity))
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	require.NoError(t, <-serverErr)
}

func TestServerRejectsWrongClientIdentity(t *testing.T) {
	ca := newTestCA(t)
	server, err := Load(testServerIdentity, ca.files(t, testServerIdentity))
	require.NoError(t, err)
	other, err := Load(testOtherIdentity, ca.files(t, testOtherIdentity))
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	tlsListener := tls.NewListener(listener, server.ServerConfig(testClientIdentity))
	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := tlsListener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer conn.Close()
		serverErr <- conn.(*tls.Conn).Handshake()
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), other.ClientConfig(testServerIdentity))
	if err == nil {
		_ = conn.Close()
	}
	serverHandshakeErr := <-serverErr
	require.Error(t, serverHandshakeErr)
	assert.Contains(t, serverHandshakeErr.Error(), "client SPIFFE ID")
}

func TestClientRejectsWrongServerIdentity(t *testing.T) {
	ca := newTestCA(t)
	otherServer, err := Load(testOtherIdentity, ca.files(t, testOtherIdentity))
	require.NoError(t, err)
	client, err := Load(testClientIdentity, ca.files(t, testClientIdentity))
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	tlsListener := tls.NewListener(listener, otherServer.ServerConfig(testClientIdentity))
	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := tlsListener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer conn.Close()
		serverErr <- conn.(*tls.Conn).Handshake()
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), client.ClientConfig(testServerIdentity))
	if err == nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server SPIFFE ID")
	<-serverErr
}

func TestBackendTransportRejectsDowngradeAndMissingIdentity(t *testing.T) {
	ca := newTestCA(t)
	client, err := Load(testOtherIdentity, ca.files(t, testOtherIdentity))
	require.NoError(t, err)

	_, err = client.BackendTransport(testClientIdentity, "http://service.local:7070")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must use https")

	_, err = client.BackendTransport("", "https://service.local:7070")
	assert.EqualError(t, err, "mTLS backend server identity is missing")
}
