package webtimeout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOrderedPathPolicies(t *testing.T) {
	config, err := Compile(&Config{Default: &Policy{Total: "30s"}, Paths: []Path{
		{Path: "/go/private/*", Total: "5s"},
		{Path: "/go/**", Total: "15m"},
		{Path: "/assets/*.zip", Total: "2m"},
		{Path: "/a/**/b", Total: "3m"},
		{Path: "/exact", Total: "1m"},
	}})
	require.NoError(t, err)
	for path, want := range map[string]time.Duration{
		"/go": 15 * time.Minute, "/go/": 15 * time.Minute, "/go/x/y": 15 * time.Minute,
		"/gopher": 30 * time.Second, "/go/private/x": 5 * time.Second,
		"/go/private/x/y": 15 * time.Minute, "/assets/a.zip": 2 * time.Minute,
		"/assets/a/b.zip": 30 * time.Second, "/a/b": 3 * time.Minute, "/a/x/y/b": 3 * time.Minute,
		"/exact": time.Minute, "/exact/": 30 * time.Second,
	} {
		require.Equal(t, want, config.Total(path), path)
	}
	fallback, err := Compile(&Config{Paths: []Path{{Path: "/**", Total: "1h"}}})
	require.NoError(t, err)
	require.Equal(t, time.Hour, fallback.Total("/"))
	require.Equal(t, time.Hour, fallback.Total("/x/y"))
	empty, err := Compile(nil)
	require.NoError(t, err)
	require.Zero(t, empty.Total("/"))
}

func TestRejectInvalidPolicies(t *testing.T) {
	for _, total := range []string{"", "0", "0s", "-1s", "15", "forever", "999999999999999999h"} {
		_, err := Compile(&Config{Default: &Policy{Total: total}})
		require.Error(t, err, total)
	}
	for _, path := range []string{"", "go/**", "/go?x=1", "/go#x", "/go/[ab]", "/foo**", "/***", "/x/../y", "/x//y", "/x\\y", "/x\ny"} {
		_, err := Compile(&Config{Paths: []Path{{Path: path, Total: "1s"}}})
		require.Error(t, err, path)
	}
}
