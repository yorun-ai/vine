package appcli

import (
	"errors"
	"strings"

	ucli "github.com/urfave/cli/v3"
)

// errIgnoreArgs reports arguments the command ignores instead of refusing, such
// as the ones a test binary receives from go test.
var errIgnoreArgs = errors.New("ignore app args")

// dropUnknownArgs removes the arguments no declared flag claims. A launcher such
// as go test passes its own arguments, and urfave/cli stops at the first flag it
// does not know: the environment variables are read after a successful parse, so
// an unknown argument left in place would also silence every configured
// environment variable and the version and help arguments.
func dropUnknownArgs(args []string, flags ...ucli.Flag) []string {
	if len(args) < 2 {
		return args
	}

	known := map[string]bool{flagLogLevel: true, flagLogRule: true}
	for _, flag := range flags {
		for _, name := range flag.Names() {
			known[name] = true
		}
	}

	kept := []string{args[0]}
	for index := 1; index < len(args); index++ {
		if args[index] == "--" {
			// Everything after the terminator is an argument, not a flag.
			return append(kept, args[index:]...)
		}
		name, carriesValue := argFlagName(args[index])
		if name == "" || known[name] {
			kept = append(kept, args[index])
			continue
		}
		// Drop the value of an unknown flag that does not carry it itself.
		if !carriesValue && index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
			index++
		}
	}
	return kept
}

// argFlagName reports the name of a flag argument and whether the argument
// carries its own value. An argument that is not a flag reports no name.
func argFlagName(arg string) (name string, carriesValue bool) {
	if len(arg) < 2 || arg[0] != '-' {
		return "", false
	}
	name = strings.TrimLeft(arg, "-")
	if name == "" {
		return "", false
	}
	if before, _, found := strings.Cut(name, "="); found {
		return before, true
	}
	return name, false
}

func isIgnorableArgsError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "flag provided but not defined:")
}
