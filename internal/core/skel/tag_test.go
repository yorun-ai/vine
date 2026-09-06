package skel

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasTagFlagMatchesIndependentFlags(t *testing.T) {
	for _, value := range []string{"sensitive", "index(0),sensitive", " sensitive , noTrim "} {
		tag := reflect.StructTag(`skel:"` + value + `"`)
		require.True(t, HasTagFlag(tag, "sensitive"))
	}
	for _, value := range []string{"", "notSensitive", "sensitiveExtra", "sensitive(false)"} {
		require.False(t, HasTagFlag(reflect.StructTag(`skel:"`+value+`"`), "sensitive"))
	}
}

func TestTagIndex(t *testing.T) {
	index, found, err := TagIndex(`skel:" sensitive , index(2) "`)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 2, index)
	_, found, err = TagIndex(`skel:"sensitive" arg:"0"`)
	require.NoError(t, err)
	require.False(t, found)
	for _, value := range []string{"index", "index()", "index(x)", "index(0", "index(0)extra", "index(0),index(1)", "index(0),index(0)", "index(999999999999999999999999)"} {
		t.Run(value, func(t *testing.T) {
			_, _, err := TagIndex(reflect.StructTag(`skel:"` + value + `"`))
			require.Error(t, err)
		})
	}
}
