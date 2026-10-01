package appcli

import (
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
)

// _VariableFlagSchema resolves only the type needed to select a CLI flag.
// Unknown paths remain YAML-valued flags; seed substitution owns validation.
type _VariableFlagSchema struct {
	data    map[string]*skel.DataSchema
	configs map[string]*skel.ConfigSchema
}

func newVariableFlagSchema(domains []*skel.DomainSchema) *_VariableFlagSchema {
	schema := new(_VariableFlagSchema{data: map[string]*skel.DataSchema{}, configs: map[string]*skel.ConfigSchema{}})
	for _, domain := range domains {
		for _, data := range domain.Data {
			schema.data[data.SkelName] = data
		}
		for _, config := range domain.Configs {
			schema.configs[config.SkelName] = config
		}
	}
	return schema
}

func (s *_VariableFlagSchema) variableType(path string) *skel.TypeSchema {
	kind := new(skel.TypeSchema{Kind: skel.TypeKindData, SkelName: "app.Vars"})
	for segment := range strings.SplitSeq(path, ".") {
		var members []*skel.MemberSchema
		switch kind.Kind {
		case skel.TypeKindData:
			data := s.data[kind.SkelName]
			if data == nil {
				return nil
			}
			members = data.Members
		case skel.TypeKindConfig:
			config := s.configs[kind.SkelName]
			if config == nil {
				return nil
			}
			members = config.Members
		case skel.TypeKindMap:
			kind = kind.Value
			continue
		default:
			return nil
		}
		kind = nil
		for _, member := range members {
			if member.Name == segment {
				kind = member.Type
				break
			}
		}
		if kind == nil {
			return nil
		}
	}
	return kind
}
