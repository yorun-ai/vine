package skel_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/core/skel"
)

func TestTagHelpers(t *testing.T) {
	require.True(t, skel.HasTagFlag(`skel:"noTrim,sensitive"`, "sensitive"))
	require.False(t, skel.HasTagFlag(`skel:"sensitiveExtra"`, "sensitive"))
	index, found, err := skel.TagIndex(`skel:"index(2),sensitive"`)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 2, index)
	_, found, err = skel.TagIndex(`arg:"0"`)
	require.NoError(t, err)
	require.False(t, found)
	_, _, err = skel.TagIndex(`skel:"index(0),index(1)"`)
	require.Error(t, err)
}
