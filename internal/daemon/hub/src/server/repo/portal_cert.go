package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
	"go.yorun.ai/vine/util/vcode"
)

type DBPortalCertRepo struct {
	Dao    *model.PortalCertDao `inject:""`
	Syncer *syncer.Syncer       `inject:""`
}

func (s *DBPortalCertRepo) ListCerts() []*core.PortalCert {
	rows := s.Dao.ListOrdered()
	certs := make([]*core.PortalCert, 0, len(rows))
	for _, row := range rows {
		certs = append(certs, toCorePortalCert(row))
	}
	return certs
}

func (s *DBPortalCertRepo) GetCertById(id int) (*core.PortalCert, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return toCorePortalCert(row), true
	}
	return nil, false
}

func (s *DBPortalCertRepo) GetCertByName(name string) (*core.PortalCert, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return toCorePortalCert(row), true
	}
	return nil, false
}

func (s *DBPortalCertRepo) SaveCert(cert *core.PortalCert) {
	row := toDBPortalCert(cert)
	s.Dao.Save(row)
	cert.Id = row.Id

	s.Syncer.SyncPortalCert(cert)
}

func (s *DBPortalCertRepo) RemoveCert(id int) bool {
	cert, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalCert(toCorePortalCert(cert))
	return true
}

func toCorePortalCert(row *model.PortalCert) *core.PortalCert {
	return &core.PortalCert{
		Id:               row.Id,
		Name:             row.Name,
		Issuer:           row.Issuer,
		Domains:          vcode.MustUnmarshalJsonS[[]string](row.Domains),
		PublicKeyBase64:  row.PublicKeyBase64,
		PrivateKeyBase64: row.PrivateKeyBase64,
		ValidFrom:        row.ValidFrom,
		ValidTo:          row.ValidTo,
	}
}

func toDBPortalCert(cert *core.PortalCert) *model.PortalCert {
	return &model.PortalCert{
		Model:            rdb.Model{Id: cert.Id},
		Name:             cert.Name,
		Issuer:           cert.Issuer,
		Domains:          vcode.MustMarshalJsonS(cert.Domains),
		PublicKeyBase64:  cert.PublicKeyBase64,
		PrivateKeyBase64: cert.PrivateKeyBase64,
		ValidFrom:        cert.ValidFrom,
		ValidTo:          cert.ValidTo,
	}
}
