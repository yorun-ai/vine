package redised

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebKeys(t *testing.T) {
	assert.Equal(t, "web:admin@demo.app:endpoint:demo.app:instance-1", FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"))
}
