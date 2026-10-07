package appcli

import (
	"fmt"
	"strconv"
	"strings"

	ucli "github.com/urfave/cli/v3"
)

// RepeatedStringFlag collects literal strings and distinguishes environment
// input from command-line input for shared variable assignment destinations.
type RepeatedStringFlag struct {
	*ucli.GenericFlag
	value *_RepeatedString
}

// NewRepeatedStringFlag collects each occurrence as one complete string, so
// commas in YAML lists, objects and text remain part of the value. Its
// environment source supplies one occurrence.
func NewRepeatedStringFlag(name string, env string, target *[]string, usage string) *RepeatedStringFlag {
	value := new(_RepeatedString{
		values: target,
	})
	return new(RepeatedStringFlag{
		GenericFlag: new(ucli.GenericFlag{
			Name:    name,
			Sources: ucli.EnvVars(env),
			Usage:   usage,
			Value:   value,
		}),
		value: value,
	})
}

// IsBoolFlag reports whether a registered variable descriptor selected a bool flag.
func (f *RepeatedStringFlag) IsBoolFlag() bool {
	return f.value.boolean
}

// TakesValue reports whether the flag requires an explicit value.
func (f *RepeatedStringFlag) TakesValue() bool {
	return !f.value.boolean
}

// PostParse loads environment values before the collected command-line values.
func (f *RepeatedStringFlag) PostParse() error {
	f.value.environment = true
	defer func() { f.value.environment = false }()
	// GenericFlag skips empty environment values. A named variable flag needs
	// to preserve an explicitly empty value, just like --name= does.
	if f.value.prefix != "" && !f.IsSet() {
		if value, _, found := f.Sources.LookupWithSource(); found && value == "" {
			if err := f.GenericFlag.Set(f.Name, value); err != nil {
				return err
			}
		}
	}
	return f.GenericFlag.PostParse()
}

type _VariableAssignments struct {
	target      *[]string
	environment []string
	commandLine []string
}

type _RepeatedString struct {
	values      *[]string
	prefix      string
	assignments *_VariableAssignments
	environment bool
	boolean     bool
	nullable    bool
}

func (v *_RepeatedString) Set(value string) error {
	if v.boolean && !(v.nullable && value == "null") {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean value")
		}
		value = strconv.FormatBool(parsed)
	}
	value = v.prefix + value
	*v.values = append(*v.values, value)
	if v.assignments != nil {
		if v.environment {
			v.assignments.environment = append(v.assignments.environment, value)
		} else {
			v.assignments.commandLine = append(v.assignments.commandLine, value)
		}
		*v.assignments.target = append(append([]string(nil), v.assignments.environment...), v.assignments.commandLine...)
	}
	return nil
}

func (v *_RepeatedString) Get() any {
	return *v.values
}

func (v *_RepeatedString) String() string {
	return strings.Join(*v.values, ", ")
}

func (v *_RepeatedString) IsBoolFlag() bool {
	return v.boolean
}
