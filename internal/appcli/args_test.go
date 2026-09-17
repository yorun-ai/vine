package appcli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	ucli "github.com/urfave/cli/v3"
)

func TestHandleIgnoresUnknownFlags(t *testing.T) {
	resetArgsForTest(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	os.Args = []string{"/tmp/vine", "-test.paniconexit0", "-test.v=true"}
	argsStdout = &stdout
	argsStderr = &stderr
	argsExit = func(int) {}

	// The environment is applied after a successful parse, so an unknown argument
	// must not reach urfave/cli: it would stop the parse and silence every
	// environment variable with it.
	endpoint := ""
	t.Setenv("VINE_TEST_ENDPOINT", "http://from-env")

	assert.NotPanics(t, func() { Handle(testFlag(&endpoint)) })
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
	assert.Equal(t, "http://from-env", endpoint)
}

func TestDropUnknownArgs(t *testing.T) {
	flags := []ucli.Flag{testFlag(new(string))}

	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "known flag keeps its argument",
			args: []string{"/tmp/vine", "--endpoint", "http://cli", "serve"},
			want: []string{"/tmp/vine", "--endpoint", "http://cli", "serve"},
		},
		{
			name: "unknown flag takes its value with it",
			args: []string{"/tmp/vine", "-test.paniconexit0", "-test.run", "TestApp", "--endpoint", "http://cli"},
			want: []string{"/tmp/vine", "--endpoint", "http://cli"},
		},
		{
			name: "unknown flag carrying its value is dropped alone",
			args: []string{"/tmp/vine", "-test.v=true", "--unknown=value", "version"},
			want: []string{"/tmp/vine", "version"},
		},
		{
			name: "arguments after the terminator stay",
			args: []string{"/tmp/vine", "--", "-test.v=true", "version"},
			want: []string{"/tmp/vine", "--", "-test.v=true", "version"},
		},
		{
			name: "single dash arguments stay",
			args: []string{"/tmp/vine", "-", "version"},
			want: []string{"/tmp/vine", "-", "version"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, dropUnknownArgs(test.args, flags...))
		})
	}
}
