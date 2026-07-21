package flag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	hubapp "go.yorun.ai/vine/internal/daemon/hub/api/app"
)

func TestNormalizeUsesDefaults(t *testing.T) {
	flags := Flag{HubEndpoint: "http://demo.local:7071"}

	flags.Normalize(false)
	assert.Equal(t, LinkDefaultAPIListen, flags.APIListen)
	assert.Equal(t, LinkDefaultIngressListen, flags.IngressListen)
	assert.Equal(t, "http://demo.local:7071", flags.HubEndpoint)
	assert.NotNil(t, flags.HubEndpointURL)
	assert.Equal(t, "demo.local", flags.HubEndpointURL.Hostname())
}

func TestNormalizeKeepsConfiguredValues(t *testing.T) {
	flags := Flag{
		APIListen:     "127.0.0.1:8100",
		IngressListen: ":8102",
		HubEndpoint:   "http://demo.local:8101",
	}

	flags.Normalize(false)
	assert.Equal(t, "127.0.0.1:8100", flags.APIListen)
	assert.Equal(t, ":8102", flags.IngressListen)
	assert.Equal(t, "http://demo.local:8101", flags.HubEndpoint)
}

func TestNormalizeClearsAPIListenInInprocMode(t *testing.T) {
	flags := Flag{
		APIListen:     "127.0.0.1:8100",
		IngressListen: ":8102",
		HubInprocMode: true,
	}

	flags.Normalize(true)
	assert.Empty(t, flags.APIListen)
	assert.Empty(t, flags.IngressListen)
	assert.Equal(t, hubapp.HubInprocEndpoint, flags.HubEndpoint)
}

func TestNormalizeRequiresHubEndpoint(t *testing.T) {
	flags := Flag{}

	assert.PanicsWithError(t, "hub-endpoint is empty", func() {
		flags.Normalize(false)
	})
}

func TestNormalizeRejectsInvalidHubEndpoint(t *testing.T) {
	flags := Flag{HubEndpoint: "://bad"}

	assert.PanicsWithError(t, "hub-endpoint is invalid: parse \"://bad\": missing protocol scheme", func() {
		flags.Normalize(false)
	})
}

func TestNormalizeRejectsHubEndpointWithoutHost(t *testing.T) {
	flags := Flag{HubEndpoint: "http:///path-only"}

	assert.PanicsWithError(t, "hub-endpoint host is empty", func() {
		flags.Normalize(false)
	})
}
