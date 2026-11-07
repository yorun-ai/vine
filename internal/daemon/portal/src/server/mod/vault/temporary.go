package vault

import (
	"container/list"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	maxTemporaryWebCertificateCacheSize = 1024
	temporaryWebCertificateLifetime     = 24 * time.Hour
	temporaryWebCertificateRenewBefore  = time.Hour
)

var errInvalidTemporaryWebCertificateHost = errors.New("invalid temporary web certificate host")

type _TemporaryWebCertificate struct {
	host string
	cert *tls.Certificate
}

type _TemporaryWebCertificates struct {
	mutex sync.Mutex
	key   *ecdsa.PrivateKey

	certs map[string]*list.Element
	order *list.List
}

func newTemporaryWebCertificates() (*_TemporaryWebCertificates, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &_TemporaryWebCertificates{
		key:   key,
		certs: make(map[string]*list.Element, maxTemporaryWebCertificateCacheSize),
		order: list.New(),
	}, nil
}

func (s *_TemporaryWebCertificates) Certificate(serverName string) (*tls.Certificate, error) {
	host, dnsNames, ipAddresses, err := temporaryWebCertificateNames(serverName)
	if err != nil {
		return nil, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	if element := s.certs[host]; element != nil {
		cached := element.Value.(*_TemporaryWebCertificate)
		if now.Before(cached.cert.Leaf.NotAfter.Add(-temporaryWebCertificateRenewBefore)) {
			s.order.MoveToBack(element)
			return cached.cert, nil
		}
		s.remove(element)
	}

	cert, err := s.issue(host, dnsNames, ipAddresses, now)
	if err != nil {
		return nil, err
	}
	element := s.order.PushBack(&_TemporaryWebCertificate{host: host, cert: cert})
	s.certs[host] = element
	if s.order.Len() > maxTemporaryWebCertificateCacheSize {
		s.remove(s.order.Front())
	}
	return cert, nil
}

func (s *_TemporaryWebCertificates) issue(
	host string,
	dnsNames []string,
	ipAddresses []net.IP,
	now time.Time,
) (*tls.Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	commonName := host
	if commonName == "" {
		commonName = "localhost"
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Vine Portal"},
			CommonName:   commonName,
		},
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(temporaryWebCertificateLifetime),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &s.key.PublicKey, s.key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  s.key,
		Leaf:        leaf,
	}, nil
}

func (s *_TemporaryWebCertificates) remove(element *list.Element) {
	entry := element.Value.(*_TemporaryWebCertificate)
	delete(s.certs, entry.host)
	s.order.Remove(element)
}

func temporaryWebCertificateNames(serverName string) (string, []string, []net.IP, error) {
	host := strings.ToLower(strings.TrimSuffix(serverName, "."))
	if host == "" {
		return "", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return host, nil, []net.IP{ip}, nil
	}
	if !isValidTemporaryWebDNSName(host) {
		return "", nil, nil, errInvalidTemporaryWebCertificateHost
	}
	return host, []string{host}, nil, nil
}

func isValidTemporaryWebDNSName(host string) bool {
	if len(host) > 253 {
		return false
	}
	for label := range strings.SplitSeq(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}
