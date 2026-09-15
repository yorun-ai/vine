package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"sort"
)

func (s *MaintenanceApiServiceServerImpl) FieldSources(kind string, name string) []skeled.FieldSource {
	var sources core.FieldSources
	switch kind {
	case seedKindAppConfig:
		if item, ok := s.AppConfigRepo.GetItemByName(name); ok {
			sources = item.FieldSources
		}
	case seedKindPortalSite:
		if item, ok := s.EntryRepo.GetEntryByName(name); ok {
			sources = item.FieldSources
		}
	case seedKindPortalRule:
		if item, ok := s.RuleRepo.GetRuleByName(name); ok {
			sources = item.FieldSources
		}
	case seedKindPortalCert:
		if item, ok := s.CertRepo.GetCertByName(name); ok {
			sources = item.FieldSources
		}
	default:
		ex.PanicNew(ex.ValidationFailed, "unknown entity kind")
	}
	result := make([]skeled.FieldSource, 0, len(sources))
	for path, source := range sources {
		result = append(result, skeled.FieldSource{Path: path, Source: source.Source, Define: source.Define, Override: source.Override, Variables: append([]string{}, source.Variables...), Template: source.Template, Bindings: fieldSourceBindings(source.Bindings)})
	}
	sort.Slice(result, func(i int, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func fieldSourceBindings(bindings []core.FieldSourceBinding) []skeled.FieldSourceBinding {
	result := make([]skeled.FieldSourceBinding, 0, len(bindings))
	for _, item := range bindings {
		result = append(result, skeled.FieldSourceBinding{Path: item.Path, Variable: item.Variable, Reference: item.Reference, Value: item.Value, DefaultUsed: item.DefaultUsed})
	}
	return result
}
