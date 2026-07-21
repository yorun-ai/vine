package testkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideConfigUsesRegisteredSkelName(t *testing.T) {
	registerTestFixtures()

	override := OverrideConfig(&_TestConfig{DSN: "sqlite://test"})

	assert.Equal(t, "testkit.TestConfig", override.Name)
	assert.Equal(t, &_TestConfig{DSN: "sqlite://test"}, override.Value)
}
