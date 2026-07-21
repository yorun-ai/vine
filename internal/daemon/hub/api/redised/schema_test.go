package redised

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaActorKeys(t *testing.T) {
	assert.Equal(t, "schema:actor:demo.user.AdminActor", FormatSchemaActorKey("demo.user.AdminActor"))
	assert.Equal(t, "schema:actor", FormatSchemaActorPrefix())
}

func TestSchemaServiceKeys(t *testing.T) {
	assert.Equal(t, "schema:service:demo.user.UserService", FormatSchemaServiceKey("demo.user.UserService"))
	assert.Equal(t, "schema:service", FormatSchemaServicePrefix())
}

func TestSchemaResourceKeys(t *testing.T) {
	assert.Equal(t, "schema:resource:demo.user.Database", FormatSchemaResourceKey("demo.user.Database"))
	assert.Equal(t, "schema:resource", FormatSchemaResourcePrefix())
}
