package appcli

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/util/vpre"
)

// flagNamePattern matches the name of one flag: lowercase letters and digits with
// dashes between them. It is the shape the declared names use, and the shape whose
// derived environment variable a shell can set.
var flagNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// FlagNames describes how an application presents the flags it declares: the
// names it accepts and discards, and the names it accepts under a different name.
// Every declared flag registers here, so Validate can report a name that no
// declared flag carries.
type FlagNames struct {
	declared   map[string]bool
	ignored    map[string]bool
	renamed    map[string]string
	registered map[string]string
}

// NewFlagNames describes the flags an application declares. ignored lists names
// the application accepts and discards, so an embedding program can own that
// parameter; renamed maps a declared name to the name the flag is registered
// under. A renamed flag drops the name and the environment variable it declared:
// the new name carries the environment variable derived from it, and a renamed
// flag cannot also be ignored.
func NewFlagNames(ignored []string, renamed map[string]string) *FlagNames {
	names := &FlagNames{
		declared:   make(map[string]bool),
		ignored:    make(map[string]bool, len(ignored)),
		renamed:    make(map[string]string, len(renamed)),
		registered: make(map[string]string),
	}
	for _, name := range ignored {
		names.ignored[name] = true
	}
	for from, to := range renamed {
		names.renamed[from] = to
	}
	return names
}

// String declares one string flag bound to target. A renamed name is registered
// under its new name, and a name in ignored stays unbound: urfave/cli reads the
// environment after a successful parse, so an unbound flag keeps both its argument
// and its environment variable out of the application.
func (n *FlagNames) String(canonical string, env string, target *string, usage string) *ucli.StringFlag {
	name, env := n.resolve(canonical, env)
	flag := &ucli.StringFlag{Name: name, Sources: ucli.EnvVars(env), Usage: usage}
	if !n.ignored[canonical] {
		flag.Destination = target
	}
	return flag
}

// Bool declares one boolean flag bound to target, with the same rules as String.
func (n *FlagNames) Bool(canonical string, env string, target *bool, usage string) *ucli.BoolFlag {
	name, env := n.resolve(canonical, env)
	flag := &ucli.BoolFlag{Name: name, Sources: ucli.EnvVars(env), Usage: usage}
	if !n.ignored[canonical] {
		flag.Destination = target
	}
	return flag
}

// Validate reports a name that no declared flag carries, and a flag that is both
// renamed and ignored.
func (n *FlagNames) Validate() {
	for _, name := range slices.Sorted(maps.Keys(n.ignored)) {
		vpre.Check(n.declared[name], "unknown flag to ignore: %q", name)
	}
	for _, name := range slices.Sorted(maps.Keys(n.renamed)) {
		vpre.Check(n.declared[name], "unknown flag to rename: %q", name)
		vpre.Check(!n.ignored[name], "flag %q cannot be renamed and ignored", name)
	}
}

// resolve records the declared name and reports the name the flag is registered
// under with the environment variable it reads. A renamed flag carries the
// variable derived from its new name, spelled the way the declared names are:
// VINE_ followed by the upper-case name with dashes as underscores.
func (n *FlagNames) resolve(canonical string, env string) (name string, envName string) {
	n.declared[canonical] = true

	name, envName = canonical, env
	if renamed, ok := n.renamed[canonical]; ok {
		name, envName = renamed, envFromName(renamed)
	}

	// A name the command line already carries cannot be declared again, and two
	// flags sharing a name leave one of them unreachable.
	vpre.Check(flagNamePattern.MatchString(name),
		"invalid flag name %q: lowercase letters and digits with dashes between them expected", name)
	vpre.Check(name != flagLogLevel && name != flagLogRule,
		"flag %q cannot be named %q: the application command line owns it", canonical, name)
	if owner, ok := n.registered[name]; ok {
		vpre.Panicf("flag %q is registered for both %q and %q", name, owner, canonical)
	}
	n.registered[name] = canonical
	return name, envName
}

// envFromName derives the environment variable of a flag name: VINE_ followed by
// the upper-case name with dashes as underscores, the spelling the declared
// variables use.
func envFromName(name string) string {
	return "VINE_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}
