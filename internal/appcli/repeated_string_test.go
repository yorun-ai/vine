package appcli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepeatedStringPreservesCommasAndEquals(t *testing.T) {
	t.Setenv("VINE_TEST_VAR", "text=from-env")
	var values []string
	_, err := parseArgs([]string{"app", "--seed-var", `origins=["a","b"]`, "--seed-var=text=a=b,c"},
		NewRepeatedStringFlag("seed-var", "VINE_TEST_VAR", &values, "variable"))
	require.NoError(t, err)
	require.Equal(t, []string{`origins=["a","b"]`, "text=a=b,c"}, values)
}

func TestRepeatedStringEnvironment(t *testing.T) {
	t.Setenv("VINE_TEST_VAR", "database={host: localhost, port: 5432}")
	var values []string
	_, err := parseArgs([]string{"app"}, NewRepeatedStringFlag("seed-var", "VINE_TEST_VAR", &values, "variable"))
	require.NoError(t, err)
	require.Equal(t, []string{"database={host: localhost, port: 5432}"}, values)
}
