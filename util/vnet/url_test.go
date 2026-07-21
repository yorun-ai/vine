package vnet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHttpURLAddsScheme(t *testing.T) {
	parsed, err := ParseHttpURL(":7099")

	require.NoError(t, err)
	assert.Equal(t, "http", parsed.Scheme)
	assert.Equal(t, "", parsed.Hostname())
	assert.Equal(t, 7099, parsed.Port())
}

func TestParseHttpURLKeepsScheme(t *testing.T) {
	parsed, err := ParseHttpURL("https://hub.example.com:8443/admin")

	require.NoError(t, err)
	assert.Equal(t, "https", parsed.Scheme)
	assert.Equal(t, "hub.example.com", parsed.Hostname())
	assert.Equal(t, 8443, parsed.Port())
	assert.Equal(t, "/admin", parsed.EscapedPath())
}

func TestParseHttpURLNormalizesEmptyPath(t *testing.T) {
	parsed, err := ParseHttpURL("https://hub.example.com:8443")

	require.NoError(t, err)
	assert.Equal(t, "/", parsed.EscapedPath())
	assert.Equal(t, "https://hub.example.com:8443/", parsed.String())
}

func TestMustParseHttpURL(t *testing.T) {
	parsed := MustParseHttpURL(":7099")

	assert.Equal(t, "http", parsed.Scheme)
	assert.Equal(t, 7099, parsed.Port())
}

func TestParseHttpURLParsesPort(t *testing.T) {
	parsed, err := ParseHttpURL("https://hub.example.com:8443/admin")

	require.NoError(t, err)
	assert.Equal(t, 8443, parsed.Port())
}

func TestParseHttpURLRejectsMissingPort(t *testing.T) {
	_, err := ParseHttpURL("https://hub.example.com/admin")

	require.EqualError(t, err, "url port is required")
}

func TestParseHttpURLRejectsInvalidPort(t *testing.T) {
	_, err := ParseHttpURL("https://hub.example.com:invalid/admin")

	require.Error(t, err)
}
