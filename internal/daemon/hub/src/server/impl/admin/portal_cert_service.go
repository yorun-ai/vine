package admin

import (
	"go.yorun.ai/vine/internal/core/skel"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalCertApiServiceServerImpl struct {
	skeled.DefaultPortalCertApiServiceServer

	PortalCertCore *core.PortalCertCore `inject:""`
}

func (s *PortalCertApiServiceServerImpl) List() []skeled.PortalCertListItem {
	certs := s.PortalCertCore.List()
	ret := make([]skeled.PortalCertListItem, 0, len(certs))
	for _, cert := range certs {
		ret = append(ret, toServerPortalCertListItem(cert))
	}
	return ret
}

func (s *PortalCertApiServiceServerImpl) Get(id int) skeled.PortalCert {
	cert := s.PortalCertCore.Get(id)
	return toServerPortalCert(cert, portalCertFieldSources(cert.FieldSources))
}

func (s *PortalCertApiServiceServerImpl) Create(creation skeled.PortalCertCreation) skeled.PortalCert {
	cert := s.PortalCertCore.Create(core.PortalCertCreation{
		Name:        creation.Name,
		Certificate: creation.Certificate,
		PrivateKey:  creation.PrivateKey,
		Enabled:     creation.Enabled,
	})
	return toServerPortalCert(cert, portalCertFieldSources(cert.FieldSources))
}

func (s *PortalCertApiServiceServerImpl) Update(id int, update skeled.PortalCertUpdate) skeled.PortalCert {
	cert := s.PortalCertCore.Update(id, core.PortalCertUpdate{
		Name:        update.Name,
		Certificate: update.Certificate,
		PrivateKey:  update.PrivateKey,
		Enabled:     update.Enabled,
	})
	return toServerPortalCert(cert, portalCertFieldSources(cert.FieldSources))
}

func (s *PortalCertApiServiceServerImpl) Remove(id int) {
	s.PortalCertCore.Remove(id)
}

func toServerPortalCert(cert *core.PortalCert, fieldSources []skeled.FieldSource) skeled.PortalCert {
	return skeled.PortalCert{
		Enabled:              cert.Enabled,
		Id:                   cert.Id,
		Name:                 cert.Name,
		Issuer:               cert.Issuer,
		Domains:              cert.Domains,
		Certificate:          cert.Certificate,
		PrivateKeyConfigured: cert.PrivateKey != "",
		ValidFrom:            skel.NewTimestamp(cert.ValidFrom),
		ValidTo:              skel.NewTimestamp(cert.ValidTo),
		FieldSources:         fieldSources,
	}
}

// toServerPortalCertListItem maps a certificate for list responses, which carry
// the entity values without its seed provenance.
func toServerPortalCertListItem(cert *core.PortalCert) skeled.PortalCertListItem {
	detail := toServerPortalCert(cert, nil)
	return skeled.PortalCertListItem{
		Enabled:              detail.Enabled,
		Id:                   detail.Id,
		Name:                 detail.Name,
		Issuer:               detail.Issuer,
		Domains:              detail.Domains,
		Certificate:          detail.Certificate,
		PrivateKeyConfigured: detail.PrivateKeyConfigured,
		ValidFrom:            detail.ValidFrom,
		ValidTo:              detail.ValidTo,
	}
}

// Certificate provenance may contain a private key in a seed template or binding.
// Keep its source labels, but never return those values through the Admin API.
func portalCertFieldSources(sources core.FieldSources) []skeled.FieldSource {
	fields := toServerFieldSources(sources)
	for i := range fields {
		// TODO: Remove only the legacy /privateKeyBase64 branch when provenance
		// migration support is retired; keep PEM private-key redaction.
		if fields[i].Path == "/privateKey" || fields[i].Path == "/privateKeyBase64" {
			fields[i].Template = nil
			fields[i].Bindings = nil
		}
	}
	return fields
}
