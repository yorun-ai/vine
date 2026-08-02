package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon"
)

func TestSPIFFEPathForUsername(t *testing.T) {
	tests := []struct {
		username string
		identity daemon.Identity
	}{
		{username: HubUsername, identity: daemon.HubIdentity},
		{username: LinkUsername, identity: daemon.LinkIdentity},
		{username: PortalUsername, identity: daemon.PortalIdentity},
	}
	for _, test := range tests {
		t.Run(test.username, func(t *testing.T) {
			path, ok := SPIFFEPathForUsername(test.username)
			require.True(t, ok)
			assert.Equal(t, test.identity.SPIFFEPath(), path)
		})
	}

	path, ok := SPIFFEPathForUsername("unknown")
	assert.False(t, ok)
	assert.Empty(t, path)
}
