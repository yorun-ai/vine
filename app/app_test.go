package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyOptionOverridesCliOption(t *testing.T) {
	cliOption := &Option{
		LinkEndpoint: "http://cli-link.local:7079",
	}

	applyOption(cliOption, Option{
		LinkEndpoint: "http://option-link.local:7079",
	})

	assert.Equal(t, "http://option-link.local:7079", cliOption.LinkEndpoint)
}

func TestApplyOptionKeepsUnsetCliOption(t *testing.T) {
	cliOption := &Option{
		LinkEndpoint: "http://cli-link.local:7079",
	}

	applyOption(cliOption, Option{})

	assert.Equal(t, "http://cli-link.local:7079", cliOption.LinkEndpoint)
}
