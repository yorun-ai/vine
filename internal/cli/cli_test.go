package cli

import (
	stdRuntime "runtime"
	"strings"
	"testing"

	"go.yorun.ai/vine/core/skel"
)

func TestRunVersion(t *testing.T) {
	result := run([]string{"version"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	expected := versionInfo().TextString() + "\n"
	if result.stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if want := "  Platform   " + stdRuntime.GOOS + "/" + stdRuntime.GOARCH + "\n"; !strings.Contains(result.stdout, want) {
		t.Fatalf("expected stdout to contain platform line %q, got %q", want, result.stdout)
	}
	if want := "  MinSkelcVersion  " + skel.MinSkelcVersion() + "\n"; !strings.Contains(result.stdout, want) {
		t.Fatalf("expected stdout to contain dependency line %q, got %q", want, result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestRunVersionJSON(t *testing.T) {
	result := run([]string{"version", "--json"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	expected := versionInfo().JSONString() + "\n"
	if result.stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}
