package logger

import (
	"log/slog"
	"testing"
)

func TestTrimSourceFileForMainModule(t *testing.T) {
	got := trimSourceFile(&slog.Source{
		File:     "/workspace/demo/impl/user_service.go",
		Function: "example.com/demo/impl.(*UserService).Ping",
	})

	if got != "impl/user_service.go" {
		t.Fatalf("unexpected main module source: got %q", got)
	}
}

func TestTrimSourceFileForReplaceDependency(t *testing.T) {
	got := trimSourceFile(&slog.Source{
		File:     "/workspace/vine/internal/app/impl.go",
		Function: "go.yorun.ai/vine/internal/app.(*appImpl).Start",
	})

	if got != "app/impl.go" {
		t.Fatalf("unexpected replace dependency source: got %q", got)
	}
}

func TestTrimSourceFileForModuleCacheDependency(t *testing.T) {
	got := trimSourceFile(&slog.Source{
		File: "/Users/test/go/pkg/mod/gorm.io/gorm@v1.23.5/callbacks.go",
	})

	if got != "gorm@v1.23.5/callbacks.go" {
		t.Fatalf("unexpected module cache source: got %q", got)
	}
}

func TestTrimSourceFileForNestedModuleCacheDependency(t *testing.T) {
	got := trimSourceFile(&slog.Source{
		File: "/Users/test/go/pkg/mod/go.yorun.ai/vine@v1.1.0-alpha9/internal/core/web/server.go",
	})

	if got != "web/server.go" {
		t.Fatalf("unexpected nested module cache source: got %q", got)
	}
}

func TestTrimSourceFileFallsBackToShortCallerFile(t *testing.T) {
	got := trimSourceFile(&slog.Source{
		File: "/tmp/random/location/file.go",
	})

	if got != "location/file.go" {
		t.Fatalf("unexpected fallback source: got %q", got)
	}
}
