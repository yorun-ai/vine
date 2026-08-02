package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"go.yorun.ai/vine/util/vpre"
)

const (
	HubIdentity    = "vine.hub"
	LinkIdentity   = "vine.link"
	PortalIdentity = "vine.portal"
)

// Files identifies the certificate material shared by a Vine component's
// backend servers and clients. All three paths must be configured together.
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

// Identity owns one component certificate and the CA used to authenticate
// other Vine backend components. A disabled identity preserves inproc and
// local-development behavior when no certificate paths are configured.
type Identity struct {
	name        string
	spiffeID    spiffeid.ID
	certificate tls.Certificate
	roots       *x509.CertPool
	bundle      *x509bundle.Bundle
	enabled     bool
	transportMu sync.Mutex
	transports  map[string]http.RoundTripper
}

func DisabledIdentity() *Identity {
	return &Identity{transports: map[string]http.RoundTripper{}}
}

func Load(identityName string, files Files) (*Identity, error) {
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
		return nil, fmt.Errorf("load %s mTLS SPIFFE identity: %w", identityName, err)
	}
	expectedPath, err := componentSPIFFEPath(identityName)
	if err != nil {
		return nil, err
	}
	if identityID.Path() != expectedPath {
		return nil, fmt.Errorf("mTLS certificate SPIFFE ID %q does not identify %s; expected path %q", identityID, identityName, expectedPath)
	}

	bundle, err := x509bundle.Parse(identityID.TrustDomain(), caPEM)
	if err != nil {
		return nil, fmt.Errorf("parse mTLS SPIFFE trust bundle: %w", err)
	}
	if bundle.Empty() {
		return nil, errors.New("mTLS CA file contains no certificates")
	}
	if err := verifyOwnCertificate(identityName, identityID, certificate, roots, bundle); err != nil {
		return nil, err
	}

	return &Identity{
		name:        identityName,
		spiffeID:    identityID,
		certificate: certificate,
		roots:       roots,
		bundle:      bundle,
		enabled:     true,
		transports:  map[string]http.RoundTripper{},
	}, nil
}

func MustLoad(identityName string, files Files) *Identity {
	identity, err := Load(identityName, files)
	vpre.CheckNilError(err, "load %s mTLS identity failed", identityName)
	return identity
}

func verifyOwnCertificate(
	identityName string,
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
		return fmt.Errorf("verify %s mTLS X.509-SVID: %w", identityName, err)
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
			return fmt.Errorf("mTLS certificate for %s is not valid for %s: %w", identityName, usageName(usage), err)
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

func (i *Identity) Name() string {
	if i == nil {
		return ""
	}
	return i.name
}

// SPIFFEID returns the exact X.509-SVID identity loaded for this component.
// Disabled identities return an empty string.
func (i *Identity) SPIFFEID() string {
	if !i.Enabled() {
		return ""
	}
	return i.spiffeID.String()
}

// ServerConfig requires a client certificate signed by the configured CA and,
// when identities are provided, restricts callers to their exact SPIFFE IDs in
// this component's trust domain.
func (i *Identity) ServerConfig(allowedClientIdentities ...string) *tls.Config {
	vpre.Check(i.Enabled(), "mTLS identity is disabled")
	allowed := make([]spiffeid.ID, 0, len(allowedClientIdentities))
	for _, identityName := range allowedClientIdentities {
		allowed = append(allowed, i.mustComponentSPIFFEID(identityName))
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
		for _, allowedID := range allowed {
			if peerID == allowedID {
				return nil
			}
		}
		return fmt.Errorf("mTLS client SPIFFE ID %q is not allowed; expected one of %v", peerID, allowed)
	}
	return config
}

func (i *Identity) ClientConfig(serverIdentity string) *tls.Config {
	vpre.Check(i.Enabled(), "mTLS identity is disabled")
	expectedID := i.mustComponentSPIFFEID(serverIdentity)
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
// exact SPIFFE ID for the requested component in this identity's trust domain.
func (i *Identity) ConnectionHasIdentity(state tls.ConnectionState, identityName string) bool {
	if !i.Enabled() {
		return false
	}
	peerID, err := i.verifyPeer(state.PeerCertificates, x509.ExtKeyUsageClientAuth)
	if err != nil {
		return false
	}
	expectedID, err := i.componentSPIFFEID(identityName)
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

func (i *Identity) mustComponentSPIFFEID(identityName string) spiffeid.ID {
	id, err := i.componentSPIFFEID(identityName)
	vpre.CheckNilError(err, "build %s SPIFFE ID failed", identityName)
	return id
}

func (i *Identity) componentSPIFFEID(identityName string) (spiffeid.ID, error) {
	path, err := componentSPIFFEPath(identityName)
	if err != nil {
		return spiffeid.ID{}, err
	}
	return spiffeid.FromPath(i.spiffeID.TrustDomain(), path)
}

func componentSPIFFEPath(identityName string) (string, error) {
	switch identityName {
	case HubIdentity, LinkIdentity, PortalIdentity:
		return "/vine/daemon/" + identityName, nil
	default:
		return "", fmt.Errorf("unknown Vine component identity %q", identityName)
	}
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
func (i *Identity) HTTPTransport(serverIdentity string) http.RoundTripper {
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
