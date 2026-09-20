package core

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/core/ex"
)

type PortalCert struct {
	FieldSources FieldSources
	Id           int
	Name         string
	Issuer       string
	Domains      []string
	Certificate  string
	PrivateKey   string
	ValidFrom    time.Time
	ValidTo      time.Time
	// Enabled decides whether Hub publishes the certificate to Portal.
	Enabled bool
}

type PortalCertCreation struct {
	Name        string
	Certificate string
	PrivateKey  string
	// Enabled is optional and defaults to true.
	Enabled *bool
}

type PortalCertUpdate struct {
	Name        *string
	Certificate *string
	PrivateKey  *string
	Enabled     *bool
}

// PortalCertRepo stores Portal certificates. List and the lookups return entities
// the caller owns.
type PortalCertRepo interface {
	List() []*PortalCert
	GetById(id int) (*PortalCert, bool)
	GetByName(name string) (*PortalCert, bool)
	Save(cert *PortalCert)
	Remove(id int) bool
}

type PortalCertCore struct {
	PortalCertRepo PortalCertRepo `inject:""`
}

func (m *PortalCertCore) List() []*PortalCert {
	return m.PortalCertRepo.List()
}

// FindByName returns the certificate with the name.
func (m *PortalCertCore) FindByName(name string) (*PortalCert, bool) {
	return m.PortalCertRepo.GetByName(name)
}

func (m *PortalCertCore) Get(id int) *PortalCert {
	cert, ok := m.PortalCertRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))
	return cert
}

func (m *PortalCertCore) Create(creation PortalCertCreation) *PortalCert {
	_, ok := m.PortalCertRepo.GetByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("entry cert %q already exists", creation.Name))

	cert := new(m.Validate(PortalCert{
		Name:        creation.Name,
		Certificate: creation.Certificate,
		PrivateKey:  creation.PrivateKey,
		Enabled:     EnabledOrDefault(creation.Enabled),
	}))
	m.PortalCertRepo.Save(cert)
	return cert
}

func (m *PortalCertCore) Update(id int, update PortalCertUpdate) *PortalCert {
	cert, ok := m.PortalCertRepo.GetById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))

	next := &PortalCert{
		FieldSources: cloneFieldSources(cert.FieldSources),
		Id:           cert.Id,
		Name:         cert.Name,
		Issuer:       cert.Issuer,
		Domains:      cert.Domains,
		Certificate:  cert.Certificate,
		PrivateKey:   cert.PrivateKey,
		ValidFrom:    cert.ValidFrom,
		ValidTo:      cert.ValidTo,
		Enabled:      cert.Enabled,
	}
	if update.Name != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/name")
		if *update.Name != cert.Name {
			_, exists := m.PortalCertRepo.GetByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("entry cert %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.Certificate != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/certificate")
		next.Certificate = *update.Certificate
	}
	if update.PrivateKey != nil {
		next.FieldSources = overrideFieldSource(next.FieldSources, "/privateKey")
		next.PrivateKey = *update.PrivateKey
	}
	if update.Enabled != nil {
		// A seed declares the switch as disabled, so it owns that source path.
		next.FieldSources = overrideFieldSource(next.FieldSources, "/disabled")
		next.Enabled = *update.Enabled
	}

	*next = m.Validate(*next)
	m.PortalCertRepo.Save(next)
	return next
}

func (m *PortalCertCore) Remove(id int) {
	ok := m.PortalCertRepo.Remove(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))
}

type _PortalCertMetadata struct {
	Issuer    string
	Domains   []string
	ValidFrom time.Time
	ValidTo   time.Time
}

func parsePortalCertMetadata(cert *x509.Certificate) _PortalCertMetadata {
	issuer := cert.Issuer.CommonName
	if issuer == "" {
		issuer = cert.Issuer.String()
	}

	domains := make([]string, 0, len(cert.DNSNames)+len(cert.IPAddresses)+1)
	domains = append(domains, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		domains = append(domains, ip.String())
	}
	if len(domains) == 0 && cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
	}

	return _PortalCertMetadata{
		Issuer:    issuer,
		Domains:   domains,
		ValidFrom: cert.NotBefore,
		ValidTo:   cert.NotAfter,
	}
}

// Validate parses the certificate and derives metadata without accessing storage.
func (*PortalCertCore) Validate(cert PortalCert) PortalCert {
	ex.PanicNewIfNot(strings.TrimSpace(cert.Name) != "", ex.OperationFailed, "certificate name is required")
	pair, err := tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, "certificate and privateKey must be a matching PEM certificate chain and private key")
	for i, der := range pair.Certificate {
		parsed, err := x509.ParseCertificate(der)
		ex.PanicNewIfNot(err == nil, ex.OperationFailed, "certificate contains an invalid X.509 certificate")
		if i == 0 {
			pair.Leaf = parsed
		}
	}
	cert.Certificate = strings.TrimSpace(cert.Certificate) + "\n"
	cert.PrivateKey = strings.TrimSpace(cert.PrivateKey) + "\n"
	metadata := parsePortalCertMetadata(pair.Leaf)
	cert.Issuer = metadata.Issuer
	cert.Domains = metadata.Domains
	cert.ValidFrom = metadata.ValidFrom
	cert.ValidTo = metadata.ValidTo
	return cert
}

// Save creates or replaces a certificate by name, preserving an existing ID.
func (m *PortalCertCore) Save(cert PortalCert) *PortalCert {
	cert = m.Validate(cert)
	cert.Id = 0
	if current, ok := m.PortalCertRepo.GetByName(cert.Name); ok {
		cert.Id = current.Id
	}
	m.PortalCertRepo.Save(&cert)
	return &cert
}
