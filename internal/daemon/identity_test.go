package daemon

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDaemonSPIFFEIdentities(t *testing.T) {
	assert.Equal(t, "vine.hub", HubIdentity.String())
	assert.Equal(t, "/vine/daemon/vine.hub", HubIdentity.SPIFFEPath().String())
	assert.Equal(t, "vine.link", LinkIdentity.String())
	assert.Equal(t, "/vine/daemon/vine.link", LinkIdentity.SPIFFEPath().String())
	assert.Equal(t, "vine.portal", PortalIdentity.String())
	assert.Equal(t, "/vine/daemon/vine.portal", PortalIdentity.SPIFFEPath().String())
	assert.Empty(t, Identity("").SPIFFEPath())
}

func TestIdentityJSONUsesApplicationName(t *testing.T) {
	encoded, err := json.Marshal(LinkIdentity)
	assert.NoError(t, err)
	assert.Equal(t, `"vine.link"`, string(encoded))

	var decoded Identity
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, LinkIdentity, decoded)
}
