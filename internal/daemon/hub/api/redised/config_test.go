package redised

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigKeys(t *testing.T) {
	assert.Equal(t, "config:demo.user.FeatureConfig", FormatConfigKey("demo.user.FeatureConfig"))
}
