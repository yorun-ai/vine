package core

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"time"
	"unicode"

	"go.yorun.ai/vine/internal/core/ex"
)

type PortalCert struct {
	Id               int
	Name             string
	Issuer           string
	Domains          []string
	PublicKeyBase64  string
	PrivateKeyBase64 string
	ValidFrom        time.Time
	ValidTo          time.Time
}

type PortalCertCreation struct {
	Name             string
	PublicKeyBase64  string
	PrivateKeyBase64 string
}

type PortalCertUpdate struct {
	Name             *string
	PublicKeyBase64  *string
	PrivateKeyBase64 *string
}

type PortalCertRepo interface {
	ListCerts() []*PortalCert
	GetCertById(id int) (*PortalCert, bool)
	GetCertByName(name string) (*PortalCert, bool)
	SaveCert(cert *PortalCert)
	RemoveCert(id int) bool
}

type PortalCertCore struct {
	PortalCertRepo PortalCertRepo `inject:""`
}

func (m *PortalCertCore) List() []*PortalCert {
	return m.PortalCertRepo.ListCerts()
}

func (m *PortalCertCore) Get(id int) *PortalCert {
	cert, ok := m.PortalCertRepo.GetCertById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))
	return cert
}

func (m *PortalCertCore) Create(creation PortalCertCreation) *PortalCert {
	_, ok := m.PortalCertRepo.GetCertByName(creation.Name)
	ex.PanicNewIfNot(!ok, ex.OperationFailed, ex.F("entry cert %q already exists", creation.Name))

	metadata := parsePortalCertMetadata(creation.PublicKeyBase64)
	cert := &PortalCert{
		Name:             creation.Name,
		Issuer:           metadata.Issuer,
		Domains:          metadata.Domains,
		PublicKeyBase64:  creation.PublicKeyBase64,
		PrivateKeyBase64: creation.PrivateKeyBase64,
		ValidFrom:        metadata.ValidFrom,
		ValidTo:          metadata.ValidTo,
	}
	m.PortalCertRepo.SaveCert(cert)
	return cert
}

func (m *PortalCertCore) Update(id int, update PortalCertUpdate) *PortalCert {
	cert, ok := m.PortalCertRepo.GetCertById(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))

	next := &PortalCert{
		Id:               cert.Id,
		Name:             cert.Name,
		Issuer:           cert.Issuer,
		Domains:          cert.Domains,
		PublicKeyBase64:  cert.PublicKeyBase64,
		PrivateKeyBase64: cert.PrivateKeyBase64,
		ValidFrom:        cert.ValidFrom,
		ValidTo:          cert.ValidTo,
	}
	if update.Name != nil {
		if *update.Name != cert.Name {
			_, exists := m.PortalCertRepo.GetCertByName(*update.Name)
			ex.PanicNewIfNot(!exists, ex.OperationFailed, ex.F("entry cert %q already exists", *update.Name))
		}
		next.Name = *update.Name
	}
	if update.PublicKeyBase64 != nil {
		next.PublicKeyBase64 = *update.PublicKeyBase64
		metadata := parsePortalCertMetadata(*update.PublicKeyBase64)
		next.Issuer = metadata.Issuer
		next.Domains = metadata.Domains
		next.ValidFrom = metadata.ValidFrom
		next.ValidTo = metadata.ValidTo
	}
	if update.PrivateKeyBase64 != nil {
		next.PrivateKeyBase64 = *update.PrivateKeyBase64
	}

	m.PortalCertRepo.SaveCert(next)
	return next
}

func (m *PortalCertCore) Remove(id int) {
	ok := m.PortalCertRepo.RemoveCert(id)
	ex.PanicNewIfNot(ok, ex.OperationFailed, ex.F("entry cert %d not found", id))
}

type _PortalCertMetadata struct {
	Issuer    string
	Domains   []string
	ValidFrom time.Time
	ValidTo   time.Time
}

func parsePortalCertMetadata(publicKeyBase64 string) _PortalCertMetadata {
	cert, err := parsePortalCert(publicKeyBase64)
	ex.PanicNewIfNot(err == nil, ex.OperationFailed, ex.F("invalid entry cert certificate: %v", err))

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

func parsePortalCert(publicKeyBase64 string) (*x509.Certificate, error) {
	value := strings.TrimSpace(publicKeyBase64)
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return x509.ParseCertificate(block.Bytes)
	}

	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	der, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		der, err = base64.RawStdEncoding.DecodeString(compact)
		if err != nil {
			return nil, err
		}
	}

	if block, _ := pem.Decode(der); block != nil {
		return x509.ParseCertificate(block.Bytes)
	}
	return x509.ParseCertificate(der)
}
