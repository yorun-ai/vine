package appcli

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	ucli "github.com/urfave/cli/v3"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vpre"
)

var variablePathSegment = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// VariableFlags registers variable paths under application-owned flag names.
// Environment inputs precede command-line inputs; command-line assignments
// retain their order across the named flags and the general seed variable flag.
func (n *FlagNames) VariableFlags(paths map[string]string, target *[]string, seed *RepeatedStringFlag) []ucli.Flag {
	return n.variableFlags(paths, target, seed, skel.RegisteredDomainDescriptors())
}

func (n *FlagNames) variableFlags(paths map[string]string, target *[]string, seed *RepeatedStringFlag, domains []*skeldesc.Domain) []ucli.Flag {
	descriptor := newVariableFlagDescriptor(domains)
	assignments := new(_VariableAssignments{
		target: target,
	})
	if seed.value.values == target {
		seed.value.values = new([]string)
		seed.value.assignments = assignments
	}
	list := make([]ucli.Flag, 0, len(paths))
	for _, path := range slices.Sorted(maps.Keys(paths)) {
		for segment := range strings.SplitSeq(path, ".") {
			vpre.Check(variablePathSegment.MatchString(segment), "seed variable %q must use camelCase path segments", path)
		}
		name := paths[path]
		env := envFromName(name)
		n.register(path, name, env)
		flag := NewRepeatedStringFlag(name, env, new([]string), "seed variable "+path+" as YAML; repeatable")
		kind := descriptor.variableType(path)
		if kind != nil && kind.Kind == skeldesc.TypeKindScalar && kind.Scalar == skeldesc.ScalarBoolean {
			flag.value.boolean = true
			flag.value.nullable = kind.Nullable
			flag.Usage = "seed boolean variable " + path + "; repeatable; bare flag means true"
		}
		flag.value.prefix = path + "="
		flag.value.assignments = assignments
		list = append(list, flag)
	}
	return list
}

// _VariableFlagDescriptor resolves only the type needed to select a CLI flag.
// Unknown paths remain YAML-valued flags; seed substitution owns validation.
type _VariableFlagDescriptor struct {
	data    map[string]*skeldesc.Data
	configs map[string]*skeldesc.Config
}

func newVariableFlagDescriptor(domains []*skeldesc.Domain) *_VariableFlagDescriptor {
	descriptor := new(_VariableFlagDescriptor{
		data:    map[string]*skeldesc.Data{},
		configs: map[string]*skeldesc.Config{},
	})
	for _, domain := range domains {
		for _, data := range domain.Data {
			descriptor.data[data.SkelName] = data
		}
		for _, config := range domain.Configs {
			descriptor.configs[config.SkelName] = config
		}
	}
	return descriptor
}

func (s *_VariableFlagDescriptor) variableType(path string) *skeldesc.Type {
	kind := new(skeldesc.Type{
		Kind:     skeldesc.TypeKindData,
		SkelName: "app.Vars",
	})
	for segment := range strings.SplitSeq(path, ".") {
		var members []*skeldesc.Member
		switch kind.Kind {
		case skeldesc.TypeKindData:
			data := s.data[kind.SkelName]
			if data == nil {
				return nil
			}
			members = data.Members
		case skeldesc.TypeKindConfig:
			config := s.configs[kind.SkelName]
			if config == nil {
				return nil
			}
			members = config.Members
		case skeldesc.TypeKindMap:
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
