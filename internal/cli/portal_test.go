package cli

import (
	"testing"

	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

func TestRunPortalServe(t *testing.T) {
	originalStart := startPortalApp
	defer func() { startPortalApp = originalStart }()

	called := false
	startPortalApp = func(flags portalflag.Flag) {
		called = true
		if flags.HubEndpoint != "http://demo.local:7071" {
			t.Fatalf("unexpected hub endpoint: %s", flags.HubEndpoint)
		}
	}

	result := run([]string{"portal", "serve", "--hub-endpoint", "http://demo.local:7071"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
	if !called {
		t.Fatal("expected portal app to start")
	}
}

func TestRunPortalServeFromEnv(t *testing.T) {
	originalStart := startPortalApp
	defer func() { startPortalApp = originalStart }()

	t.Setenv(envPortalHubEndpoint, "http://env.demo.local:8071")

	called := false
	startPortalApp = func(flags portalflag.Flag) {
		called = true
		if flags.HubEndpoint != "http://env.demo.local:8071" {
			t.Fatalf("unexpected hub endpoint: %s", flags.HubEndpoint)
		}
	}

	result := run([]string{"portal", "serve"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if !called {
		t.Fatal("expected portal app to start")
	}
}
