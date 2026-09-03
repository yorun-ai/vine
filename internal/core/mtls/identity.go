package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"go.yorun.ai/vine/util/vpre"
)

// SPIFFEPath is the path component of a workload's SPIFFE ID. The trust domain
// is taken from the loaded X.509-SVID and is never supplied by callers.
type SPIFFEPath string

// String returns the SPIFFE workload path.
func (p SPIFFEPath) String() string {
	return string(p)
}

// Files identifies the certificate material shared by a workload's servers
// and clients. All three paths must be configured together.
type Files struct {
	CAFile   string
	CertFile string
	KeyFile  string
}

func (f Files) Enabled() bool {
	return f.CAFile != "" || f.CertFile != "" || f.KeyFile != ""
}

func (f Files) Validate() error {
	if !f.Enabled() {
		return nil
	}
	if f.CAFile == "" || f.CertFile == "" || f.KeyFile == "" {
		return errors.New("mtls-ca-file, mtls-cert-file, and mtls-key-file must be configured together")
	}
	return nil
}

// Identity owns one workload certificate and the CA used to authenticate its
// peers. A disabled identity preserves local-development behavior when no
// certificate paths are configured.
type Identity struct {
	spiffeID    spiffeid.ID
	certificate tls.Certificate
	roots       *x509.CertPool
	bundle      *x509bundle.Bundle
	enabled     bool
	transportMu sync.Mutex
	transports  map[SPIFFEPath]http.RoundTripper
}

func DisabledIdentity() *Identity {
	return &Identity{transports: map[SPIFFEPath]http.RoundTripper{}}
}

func Load(identityPath SPIFFEPath, files Files) (*Identity, error) {
	if err := files.Validate(); err != nil {
		return nil, err
	}
	if !files.Enabled() {
		return DisabledIdentity(), nil
	}

	caPEM, err := os.ReadFile(files.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read mTLS CA file: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("mTLS CA file contains no certificates")
	}

	certificate, err := tls.LoadX509KeyPair(files.CertFile, files.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load mTLS certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse mTLS certificate: %w", err)
	}
	certificate.Leaf = leaf
	identityID, err := x509svid.IDFromCert(leaf)
	if err != nil {
		return nil, fmt.Errorf("load mTLS SPIFFE identity %q: %w", identityPath, err)
	}
	expectedID, err := spiffeid.FromPath(identityID.TrustDomain(), identityPath.String())
	if err != nil {
		return nil, fmt.Errorf("build expected mTLS SPIFFE ID from path %q: %w", identityPath, err)
	}
	if identityID != expectedID {
		return nil, fmt.Errorf("mTLS certificate SPIFFE ID %q does not match expected ID %q", identityID, expectedID)
	}

	bundle, err := x509bundle.Parse(identityID.TrustDomain(), caPEM)
	if err != nil {
		return nil, fmt.Errorf("parse mTLS SPIFFE trust bundle: %w", err)
	}
	if bundle.Empty() {
		return nil, errors.New("mTLS CA file contains no certificates")
	}
	if err := verifyOwnCertificate(identityPath, identityID, certificate, roots, bundle); err != nil {
		return nil, err
	}

	return &Identity{
		spiffeID:    identityID,
		certificate: certificate,
		roots:       roots,
		bundle:      bundle,
		enabled:     true,
		transports:  map[SPIFFEPath]http.RoundTripper{},
	}, nil
}

func MustLoad(identityPath SPIFFEPath, files Files) *Identity {
	identity, err := Load(identityPath, files)
	vpre.CheckNilError(err, "load mTLS identity %q failed", identityPath)
	return identity
}

func verifyOwnCertificate(
	identityPath SPIFFEPath,
	identityID spiffeid.ID,
	certificate tls.Certificate,
	roots *x509.CertPool,
	bundle *x509bundle.Bundle,
) error {
	leaf := certificate.Leaf
	certificates, err := parseCertificateChain(certificate.Certificate)
	if err != nil {
		return err
	}
	verifiedID, _, err := x509svid.Verify(certificates, bundle)
	if err != nil {
		return fmt.Errorf("verify mTLS X.509-SVID %q: %w", identityPath, err)
	}
	if verifiedID != identityID {
		return fmt.Errorf("verified mTLS SPIFFE ID %q does not match certificate identity %q", verifiedID, identityID)
	}

	intermediates := x509.NewCertPool()
	for _, cert := range certificates[1:] {
		intermediates.AddCert(cert)
	}
	for _, usage := range []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth} {
		if _, err := leaf.Verify(x509.VerifyOptions{
			Roots:         roots,
			Intermediates: intermediates,
			KeyUsages:     []x509.ExtKeyUsage{usage},
		}); err != nil {
			return fmt.Errorf("mTLS certificate for %q is not valid for %s: %w", identityPath, usageName(usage), err)
		}
	}
	return nil
}

func usageName(usage x509.ExtKeyUsage) string {
	if usage == x509.ExtKeyUsageServerAuth {
		return "server authentication"
	}
	return "client authentication"
}

func (i *Identity) Enabled() bool {
	return i != nil && i.enabled
}

// SPIFFEID returns the exact X.509-SVID identity loaded for this workload.
// Disabled identities return an empty string.
func (i *Identity) SPIFFEID() string {
	if !i.Enabled() {
		return ""
	}
	return i.spiffeID.String()
}

// ServerConfig requires a client certificate signed by the configured CA and,
// when identities are provided, restricts callers to their exact SPIFFE IDs in
// this workload's trust domain.
func (i *Identity) ServerConfig(allowedClientPaths ...SPIFFEPath) *tls.Config {
	vpre.Check(i.Enabled(), "mTLS identity is disabled")
	allowed := make([]spiffeid.ID, 0, len(allowedClientPaths))
	for _, identityPath := range allowedClientPaths {
		allowed = append(allowed, i.mustSPIFFEID(identityPath))
	}
	config := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{i.certificate},
		ClientCAs:    i.roots,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	config.VerifyConnection = func(state tls.ConnectionState) error {
		peerID, err := i.verifyPeer(state.PeerCertificates, x509.ExtKeyUsageClientAuth)
		if err != nil {
			return fmt.Errorf("verify mTLS client X.509-SVID: %w", err)
		}
		if len(allowed) == 0 {
			return nil
		}
		if slices.Contains(allowed, peerID) {
			return nil
		}
		return fmt.Errorf("mTLS client SPIFFE ID %q is not allowed; expected one of %v", peerID, allowed)
	}
	return config
}

func (i *Identity) ClientConfig(serverIdentity SPIFFEPath) *tls.Config {
	vpre.Check(i.Enabled(), "mTLS identity is disabled")
	expectedID := i.mustSPIFFEID(serverIdentity)
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{i.certificate},
		InsecureSkipVerify: true, // URI SAN verification is performed below, including the full certificate chain.
		VerifyConnection: func(state tls.ConnectionState) error {
			peerID, err := i.verifyPeer(state.PeerCertificates, x509.ExtKeyUsageServerAuth)
			if err != nil {
				return fmt.Errorf("verify mTLS server X.509-SVID: %w", err)
			}
			if peerID != expectedID {
				return fmt.Errorf("mTLS server SPIFFE ID %q is not allowed; expected %q", peerID, expectedID)
			}
			return nil
		},
	}
}

// ConnectionHasIdentity reports whether a verified TLS peer presents the
// exact requested SPIFFE ID in this identity's trust domain.
func (i *Identity) ConnectionHasIdentity(state tls.ConnectionState, identityPath SPIFFEPath) bool {
	if !i.Enabled() {
		return false
	}
	peerID, err := i.verifyPeer(state.PeerCertificates, x509.ExtKeyUsageClientAuth)
	if err != nil {
		return false
	}
	expectedID, err := i.spiffeIDFromPath(identityPath)
	return err == nil && peerID == expectedID
}

func (i *Identity) verifyPeer(certificates []*x509.Certificate, usage x509.ExtKeyUsage) (spiffeid.ID, error) {
	if len(certificates) == 0 {
		return spiffeid.ID{}, errors.New("peer certificate is missing")
	}
	peerID, _, err := x509svid.Verify(certificates, i.bundle)
	if err != nil {
		return spiffeid.ID{}, err
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certificates[1:] {
		intermediates.AddCert(cert)
	}
	if _, err := certificates[0].Verify(x509.VerifyOptions{
		Roots:         i.roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{usage},
	}); err != nil {
		return spiffeid.ID{}, fmt.Errorf("SPIFFE certificate is not valid for %s: %w", usageName(usage), err)
	}
	return peerID, nil
}

func (i *Identity) mustSPIFFEID(identityPath SPIFFEPath) spiffeid.ID {
	id, err := i.spiffeIDFromPath(identityPath)
	vpre.CheckNilError(err, "build SPIFFE ID from path %q failed", identityPath)
	return id
}

func (i *Identity) spiffeIDFromPath(identityPath SPIFFEPath) (spiffeid.ID, error) {
	return spiffeid.FromPath(i.spiffeID.TrustDomain(), identityPath.String())
}

func parseCertificateChain(rawCertificates [][]byte) ([]*x509.Certificate, error) {
	certificates := make([]*x509.Certificate, 0, len(rawCertificates))
	for _, rawCertificate := range rawCertificates {
		certificate, err := x509.ParseCertificate(rawCertificate)
		if err != nil {
			return nil, fmt.Errorf("parse mTLS certificate chain: %w", err)
		}
		certificates = append(certificates, certificate)
	}
	return certificates, nil
}

// HTTPTransport creates a transport for one expected backend identity. When
// mTLS is enabled, plaintext HTTP is rejected instead of silently downgrading.
// Disabled identities retain the existing h2c transport for development.
func (i *Identity) HTTPTransport(serverIdentity SPIFFEPath) http.RoundTripper {
	if i == nil {
		return newH2CTransport()
	}
	i.transportMu.Lock()
	defer i.transportMu.Unlock()
	if transport := i.transports[serverIdentity]; transport != nil {
		return transport
	}
	var transport http.RoundTripper
	if !i.Enabled() {
		transport = newH2CTransport()
	} else {
		transport = &http.Transport{
			ForceAttemptHTTP2:  true,
			DisableCompression: true,
			TLSClientConfig:    i.ClientConfig(serverIdentity),
		}
	}
	i.transports[serverIdentity] = transport
	return transport
}
