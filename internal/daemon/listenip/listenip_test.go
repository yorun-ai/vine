package listenip

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNormalize(t *testing.T) {
	ips, err := Normalize([]string{" ::1 ", "127.0.0.1", "::ffff:127.0.0.1", "0:0:0:0:0:0:0:1"})
	require.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1", "::1"}, ips)
	empty, err := Normalize(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
	for _, ips := range [][]string{{""}, {"localhost"}, {"127.0.0.1:80"}, {"[::1]"}, {"fe80::1%lo0"}, {"0.0.0.0", "127.0.0.1"}, {"::", "::1"}} {
		_, err := Normalize(ips)
		assert.Error(t, err, "%v", ips)
	}
	_, err = Normalize([]string{"0.0.0.0", "::"})
	require.NoError(t, err)
}

func TestBindingAndOverlap(t *testing.T) {
	for _, test := range []struct {
		ip      string
		network string
		address string
	}{
		{"", "tcp", "0.0.0.0:8080"},
		{"127.0.0.1", "tcp4", "127.0.0.1:8080"},
		{"::1", "tcp6", "[::1]:8080"},
	} {
		network, address := Binding(test.ip, 8080)
		assert.Equal(t, test.network, network)
		assert.Equal(t, test.address, address)
	}
	assert.True(t, Overlap("", "::1"))
	assert.True(t, Overlap("0.0.0.0", "127.0.0.1"))
	assert.True(t, Overlap("::", "::1"))
	assert.False(t, Overlap("0.0.0.0", "::"))
	assert.False(t, Overlap("127.0.0.1", "127.0.0.2"))
}
