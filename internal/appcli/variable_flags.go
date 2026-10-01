package appcli

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/util/vpre"
)

var variablePathSegment = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// VariableFlags registers variable paths under application-owned flag names.
// Environment inputs precede command-line inputs; command-line assignments
// retain their order across the named flags and the general seed variable flag.
func (n *FlagNames) VariableFlags(paths map[string]string, target *[]string, seed *RepeatedStringFlag) []ucli.Flag {
	assignments := new(_VariableAssignments{target: target})
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
		env := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		n.register(path, name, env)
		flag := NewRepeatedStringFlag(name, env, new([]string), "seed variable "+path+" as YAML; repeatable")
		flag.value.prefix = path + "="
		flag.value.assignments = assignments
		list = append(list, flag)
	}
	return list
}
