package vault

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"time"

	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

type _Certificate struct {
	name      string
	domains   []string
	validFrom time.Time
	validTo   time.Time
	cert      *tls.Certificate
}

func newCertificate(cert *redised.PortalCert) (*_Certificate, error) {
	publicKeyPEM, err := base64.StdEncoding.DecodeString(cert.PublicKeyBase64)
	if err != nil {
		return nil, err
	}
	privateKeyPEM, err := base64.StdEncoding.DecodeString(cert.PrivateKeyBase64)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(publicKeyPEM, privateKeyPEM)
	if err != nil {
		return nil, err
	}
	if tlsCert.Leaf == nil && len(tlsCert.Certificate) > 0 {
		tlsCert.Leaf, err = x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			return nil, err
		}
	}

	return &_Certificate{
		name:      cert.Name,
		domains:   certificateDomains(tlsCert.Leaf),
		validFrom: cert.ValidFrom,
		validTo:   cert.ValidTo,
		cert:      &tlsCert,
	}, nil
}

func certificateDomains(cert *x509.Certificate) []string {
	ret := make([]string, 0, len(cert.DNSNames)+len(cert.IPAddresses)+1)
	for _, domain := range cert.DNSNames {
		ret = append(ret, strings.ToLower(domain))
	}
	for _, ip := range cert.IPAddresses {
		ret = append(ret, ip.String())
	}
	if len(ret) == 0 && cert.Subject.CommonName != "" {
		ret = append(ret, strings.ToLower(cert.Subject.CommonName))
	}
	return ret
}

func (c *_Certificate) MatchesWildcardHost(host string) bool {
	for _, domain := range c.domains {
		if matchesWildcardCertDomain(domain, host) {
			return true
		}
	}
	return false
}

func matchesWildcardCertDomain(domain string, host string) bool {
	if !strings.HasPrefix(domain, "*.") {
		return false
	}
	suffix := domain[1:]
	return strings.HasSuffix(host, suffix) && strings.Count(host, ".") == strings.Count(domain, ".")
}
