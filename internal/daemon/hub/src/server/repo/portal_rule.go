package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/mod/syncer"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"
	"go.yorun.ai/vine/internal/infra/rdb"
)

type DBPortalRuleRepo struct {
	Dao    *model.PortalRuleDao `inject:""`
	Syncer *syncer.Syncer       `inject:""`
}

func (s *DBPortalRuleRepo) ListRules() []core.PortalRule {
	rows := s.Dao.ListOrdered()
	rules := make([]core.PortalRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, *toCorePortalRule(row))
	}
	return rules
}

func (s *DBPortalRuleRepo) GetRuleById(id int) (*core.PortalRule, bool) {
	if row, ok := s.Dao.ById(id); ok {
		return toCorePortalRule(row), true
	}
	return nil, false
}

func (s *DBPortalRuleRepo) GetRuleByName(name string) (*core.PortalRule, bool) {
	if row, ok := s.Dao.ByName(name); ok {
		return toCorePortalRule(row), true
	}
	return nil, false
}

func (s *DBPortalRuleRepo) SaveRule(rule *core.PortalRule) {
	row := toDBPortalRule(rule)
	s.Dao.Save(row)
	rule.Id = row.Id

	s.Syncer.SyncPortalRule(rule)
}

func (s *DBPortalRuleRepo) RemoveRule(id int) bool {
	rule, ok := s.Dao.DeleteById(id)
	if !ok {
		return false
	}
	s.Syncer.RemovePortalRule(toCorePortalRule(rule))
	return true
}

func toCorePortalRule(row *model.PortalRule) *core.PortalRule {
	return &core.PortalRule{
		Id:                 row.Id,
		Name:               row.Name,
		Scheme:             row.Scheme,
		Host:               row.Host,
		Port:               row.Port,
		PathPrefix:         row.PathPrefix,
		TargetType:         row.TargetType,
		SiteName:           row.SiteName,
		RedirectionPattern: row.RedirectionPattern,
		BuiltIn:            row.BuiltIn,
	}
}

func toDBPortalRule(rule *core.PortalRule) *model.PortalRule {
	return &model.PortalRule{
		Model:              rdb.Model{Id: rule.Id},
		Name:               rule.Name,
		Scheme:             rule.Scheme,
		Host:               rule.Host,
		Port:               rule.Port,
		PathPrefix:         rule.PathPrefix,
		TargetType:         rule.TargetType,
		SiteName:           rule.SiteName,
		RedirectionPattern: rule.RedirectionPattern,
		BuiltIn:            rule.BuiltIn,
	}
}
