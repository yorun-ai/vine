package admin

import (
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"sort"
)

// toServerFieldSources maps the origin of every entity field that carries seed
// information; entities without seed origins return an empty list.
func toServerFieldSources(sources core.FieldSources) []skeled.FieldSource {
	result := make([]skeled.FieldSource, 0, len(sources))
	for path, source := range sources {
		result = append(result, skeled.FieldSource{
			Path:      path,
			Source:    source.Source,
			Define:    source.Define,
			Override:  source.Override,
			Variables: append([]string{}, source.Variables...),
			Template:  source.Template,
			Bindings:  fieldSourceBindings(source.Bindings),
		})
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
