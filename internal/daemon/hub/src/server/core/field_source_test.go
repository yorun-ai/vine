package core

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceOverrideDropsReplacedDescendants(t *testing.T) {
	original := FieldSources{"/value/removed": {Source: "domain/booker", Define: "domain/booker", Variables: []string{"OLD"}}, "/name": {Source: "app/default", Define: "app/default"}}
	next := overrideFieldSource(cloneFieldSources(original), "/value")
	require.Contains(t, original, "/value/removed")
	require.NotContains(t, next, "/value/removed")
	require.Equal(t, FieldSource{Source: "hub", Override: "hub"}, next["/value"])
	require.Equal(t, original["/name"], next["/name"])
}

func TestHubOverrideUpdatesSourceAndPreservesDefine(t *testing.T) {
	sources := FieldSources{"/value/enabled": {Source: "profile/dev", Define: "domain/booker", Override: "profile/dev"}}
	next := overrideFieldSource(sources, "/value/enabled")
	require.Equal(t, FieldSource{Source: "hub", Define: "domain/booker", Override: "hub"}, next["/value/enabled"])
}
