package impl

import (
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalCertServiceServerImpl struct {
	skeled.DefaultPortalCertServiceServer

	PortalCertCore *core.PortalCertCore `inject:""`
}

func (s *PortalCertServiceServerImpl) List() []skeled.PortalCert {
	certs := s.PortalCertCore.List()
	ret := make([]skeled.PortalCert, 0, len(certs))
	for _, cert := range certs {
		ret = append(ret, toServerPortalCert(cert))
	}
	return ret
}

func (s *PortalCertServiceServerImpl) Get(id int) skeled.PortalCert {
	return toServerPortalCert(s.PortalCertCore.Get(id))
}

func (s *PortalCertServiceServerImpl) Create(creation skeled.PortalCertCreation) skeled.PortalCert {
	return toServerPortalCert(s.PortalCertCore.Create(core.PortalCertCreation{
		Name:             creation.Name,
		PublicKeyBase64:  creation.PublicKeyBase64,
		PrivateKeyBase64: creation.PrivateKeyBase64,
	}))
}

func (s *PortalCertServiceServerImpl) Update(id int, update skeled.PortalCertUpdate) skeled.PortalCert {
	return toServerPortalCert(s.PortalCertCore.Update(id, core.PortalCertUpdate{
		Name:             update.Name,
		PublicKeyBase64:  update.PublicKeyBase64,
		PrivateKeyBase64: update.PrivateKeyBase64,
	}))
}

func (s *PortalCertServiceServerImpl) Remove(id int) {
	s.PortalCertCore.Remove(id)
}

func toServerPortalCert(cert *core.PortalCert) skeled.PortalCert {
	return skeled.PortalCert{
		Id:                   cert.Id,
		Name:                 cert.Name,
		Issuer:               cert.Issuer,
		Domains:              cert.Domains,
		PublicKeyBase64:      cert.PublicKeyBase64,
		PrivateKeyConfigured: cert.PrivateKeyBase64 != "",
		ValidFrom:            skel.NewTimestamp(cert.ValidFrom),
		ValidTo:              skel.NewTimestamp(cert.ValidTo),
	}
}
