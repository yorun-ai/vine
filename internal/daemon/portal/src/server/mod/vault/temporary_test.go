package vault

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemporaryWebCertificatesEvictLeastRecentlyUsedCert(t *testing.T) {
	store, err := newTemporaryWebCertificates()
	require.NoError(t, err)
	first, err := store.Certificate("first.local")
	require.NoError(t, err)

	for index := 1; index < maxTemporaryWebCertificateCacheSize; index++ {
		_, err = store.Certificate(fmt.Sprintf("host-%d.local", index))
		require.NoError(t, err)
	}
	refreshedFirst, err := store.Certificate("first.local")
	require.NoError(t, err)
	assert.Same(t, first, refreshedFirst)

	_, err = store.Certificate("overflow.local")
	require.NoError(t, err)
	assert.Len(t, store.certs, maxTemporaryWebCertificateCacheSize)
	assert.NotContains(t, store.certs, "host-1.local")
	assert.Contains(t, store.certs, "first.local")
}

func TestTemporaryWebCertificatesNormalizeServerName(t *testing.T) {
	store, err := newTemporaryWebCertificates()
	require.NoError(t, err)

	first, err := store.Certificate("DASHBOARD.LOCAL.")
	require.NoError(t, err)
	second, err := store.Certificate("dashboard.local")
	require.NoError(t, err)

	assert.Same(t, first, second)
}
