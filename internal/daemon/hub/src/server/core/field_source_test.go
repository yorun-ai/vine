package core

import (
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
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

func TestOverrideClearsTemplateAndBindingsWithoutMutatingOriginal(t *testing.T) {
	template := skel.JSON(`"${enabled}"`)
	original := FieldSources{"/value/enabled": {Source: "app/default", Define: "app/default", Template: &template, Variables: []string{"enabled"}, Bindings: []FieldSourceBinding{{Variable: "enabled", Reference: "${enabled}", Value: skel.JSON(`false`)}}}}
	cloned := cloneFieldSources(original)
	*cloned["/value/enabled"].Template = skel.JSON(`"changed"`)
	cloned["/value/enabled"].Bindings[0].Value = skel.JSON(`true`)
	require.Equal(t, skel.JSON(`"${enabled}"`), *original["/value/enabled"].Template)
	require.Equal(t, skel.JSON(`false`), original["/value/enabled"].Bindings[0].Value)
	updated := overrideFieldSource(cloned, "/value/enabled")
	require.Nil(t, updated["/value/enabled"].Template)
	require.Empty(t, updated["/value/enabled"].Bindings)
	require.Empty(t, updated["/value/enabled"].Variables)
}
