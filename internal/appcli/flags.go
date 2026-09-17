package appcli

import (
	"maps"
	"slices"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/util/vpre"
)

// IgnoredFlagNames turns the flag names an application accepts and discards into
// the set the flag constructors consult.
func IgnoredFlagNames(names []string) map[string]bool {
	ignored := make(map[string]bool, len(names))
	for _, name := range names {
		ignored[name] = true
	}
	return ignored
}

// StringFlag declares one string flag bound to target. A name in ignored leaves
// the flag unbound: urfave/cli reads the environment after a successful parse, so
// an unbound flag keeps both its argument and its environment variable out of the
// application.
func StringFlag(name string, env string, ignored map[string]bool, target *string, usage string) *ucli.StringFlag {
	flag := &ucli.StringFlag{Name: name, Sources: ucli.EnvVars(env), Usage: usage}
	if !ignored[name] {
		flag.Destination = target
	}
	return flag
}

// BoolFlag declares one boolean flag bound to target, or unbound when the name is
// ignored.
func BoolFlag(name string, env string, ignored map[string]bool, target *bool, usage string) *ucli.BoolFlag {
	flag := &ucli.BoolFlag{Name: name, Sources: ucli.EnvVars(env), Usage: usage}
	if !ignored[name] {
		flag.Destination = target
	}
	return flag
}

// ValidateIgnoredFlags panics for an ignored name no declared flag carries, so an
// application cannot name a flag that does not exist.
func ValidateIgnoredFlags(ignored map[string]bool, flags ...ucli.Flag) {
	known := make(map[string]bool, len(flags))
	for _, flag := range flags {
		for _, name := range flag.Names() {
			known[name] = true
		}
	}
	for _, name := range slices.Sorted(maps.Keys(ignored)) {
		vpre.Check(known[name], "unknown flag to ignore: %q", name)
	}
}
