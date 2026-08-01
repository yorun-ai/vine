package runtime_test

import (
	"strings"
	"testing"

	"go.yorun.ai/vine/core/meta"
	vineruntime "go.yorun.ai/vine/core/runtime"
)

func TestFacadeExposesRuntimeMetadata(t *testing.T) {
	app := vineruntime.Application()
	if !meta.IsValidName(app.Name()) {
		t.Fatalf("invalid runtime application name: %q", app.Name())
	}
	if !meta.IsValidVersion(app.Version()) {
		t.Fatalf("invalid runtime application version: %q", app.Version())
	}
	if !meta.IsValidInstanceId(app.InstanceId()) {
		t.Fatalf("invalid runtime application instance ID: %q", app.InstanceId())
	}
	if vineruntime.GolangVersion() == "" || vineruntime.GolangCompiler() == "" {
		t.Fatal("expected Go build metadata")
	}
	if !strings.Contains(vineruntime.GolangPlatform(), "/") {
		t.Fatalf("unexpected Go platform: %q", vineruntime.GolangPlatform())
	}
	if !strings.Contains(vineruntime.Inspect(), app.Name()) {
		t.Fatal("expected inspection output to include the application name")
	}
}
