package watched

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescriptorActorKeys(t *testing.T) {
	assert.Equal(t, "descriptor:actor:demo.user.AdminActor", FormatDescriptorActorKey("demo.user.AdminActor"))
	assert.Equal(t, "descriptor:actor", FormatDescriptorActorPrefix())
}

func TestDescriptorServiceKeys(t *testing.T) {
	assert.Equal(t, "descriptor:service:demo.user.UserService", FormatDescriptorServiceKey("demo.user.UserService"))
	assert.Equal(t, "descriptor:service", FormatDescriptorServicePrefix())
}

func TestDescriptorResourceKeys(t *testing.T) {
	assert.Equal(t, "descriptor:resource:demo.user.Database", FormatDescriptorResourceKey("demo.user.Database"))
	assert.Equal(t, "descriptor:resource", FormatDescriptorResourcePrefix())
}
