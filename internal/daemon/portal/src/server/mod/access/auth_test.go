package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestParseCredentialFromAuthorizationMapsFieldsCaseInsensitively(t *testing.T) {
	credential, ok := parseCredential(testCredentialSchema(), "Key1 token123, key2 dXNlcjpwd2Q=")

	require.True(t, ok)
	assert.Equal(t, map[string]string{
		"key1": "token123",
		"key2": "dXNlcjpwd2Q=",
	}, credential)
}

func TestParseCredentialFromAuthorizationRejectsUnknownField(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1 token123, unknown value")

	assert.False(t, ok)
}

func TestParseCredentialFromAuthorizationRejectsMissingField(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1 token123")

	assert.False(t, ok)
}

func TestParseCredentialFromAuthorizationRejectsBadItem(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "Key1")

	assert.False(t, ok)
}

func TestParseCredentialRejectsEmptyCredentialSchema(t *testing.T) {
	_, ok := parseCredential(&skel.DataSchema{}, "")

	assert.False(t, ok)
}

func TestParseCredentialRejectsAllEmptyCredentialValues(t *testing.T) {
	_, ok := parseCredential(testCredentialSchema(), "key1 , key2 ")

	assert.False(t, ok)
}

func testCredentialSchema() *skel.DataSchema {
	return &skel.DataSchema{
		SkelName: "demo.UserCredential",
		Members: []*skel.MemberSchema{
			{Name: "key1"},
			{Name: "key2"},
		},
	}
}
