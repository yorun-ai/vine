package cli

import (
	"testing"

	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

func TestRunLinkServe(t *testing.T) {
	var gotFlags linkflag.Flag
	oldStartLinkApp := startLinkApp
	startLinkApp = func(flags linkflag.Flag) {
		gotFlags = flags
	}
	defer func() {
		startLinkApp = oldStartLinkApp
	}()

	result := run([]string{"link", "serve", "--api-listen", ":7088", "--ingress-listen", "127.0.0.1:7087", "--hub-endpoint", "http://demo.local:7091"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
	if gotFlags.APIListen != ":7088" {
		t.Fatalf("unexpected api listen: %q", gotFlags.APIListen)
	}
	if gotFlags.IngressListen != "127.0.0.1:7087" {
		t.Fatalf("unexpected ingress listen: %q", gotFlags.IngressListen)
	}
	if gotFlags.HubEndpoint != "http://demo.local:7091" {
		t.Fatalf("unexpected hub endpoint: %q", gotFlags.HubEndpoint)
	}
}

func TestRunLinkServeLeavesIngressListenRandomByDefault(t *testing.T) {
	var gotFlags linkflag.Flag
	oldStartLinkApp := startLinkApp
	startLinkApp = func(flags linkflag.Flag) {
		gotFlags = flags
	}
	defer func() {
		startLinkApp = oldStartLinkApp
	}()

	result := run([]string{"link", "serve", "--hub-endpoint", "http://demo.local:7091"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if gotFlags.IngressListen != "0.0.0.0:0" {
		t.Fatalf("unexpected ingress listen default: %q", gotFlags.IngressListen)
	}
}

func TestRunLinkServeFromEnv(t *testing.T) {
	var gotFlags linkflag.Flag
	oldStartLinkApp := startLinkApp
	startLinkApp = func(flags linkflag.Flag) {
		gotFlags = flags
	}
	defer func() {
		startLinkApp = oldStartLinkApp
	}()

	t.Setenv(EnvLinkAPIListen, ":8088")
	t.Setenv(EnvLinkIngressListen, "127.0.0.1:8087")
	t.Setenv(EnvLinkHubEndpoint, "http://env.demo.local:8091")

	result := run([]string{"link", "serve"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if gotFlags.APIListen != ":8088" {
		t.Fatalf("unexpected api listen: %q", gotFlags.APIListen)
	}
	if gotFlags.IngressListen != "127.0.0.1:8087" {
		t.Fatalf("unexpected ingress listen: %q", gotFlags.IngressListen)
	}
	if gotFlags.HubEndpoint != "http://env.demo.local:8091" {
		t.Fatalf("unexpected hub endpoint: %q", gotFlags.HubEndpoint)
	}
}
