package redact

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/logger"
)

type sanitizedValue struct {
	Name  string `json:"name"`
	Token string `json:"token" skel:"sensitive"`
}

func TestRenderMasksExplicitlySensitiveRoot(t *testing.T) {
	result, err := Render(map[string]string{"value": "secret"}, Option{RootSensitive: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted || result.JSON != `"<redacted>"` {
		t.Fatalf("unexpected rendering: %#v", result)
	}

	revealed, err := Render(map[string]string{"value": "visible"}, Option{
		RootSensitive:   true,
		RevealSensitive: true,
	})
	if err != nil {
		t.Fatalf("render revealed: %v", err)
	}
	if revealed.Redacted || revealed.JSON != `{"value":"visible"}` {
		t.Fatalf("unexpected revealed rendering: %#v", revealed)
	}
}

func TestRenderAppliesSanitizerBeforeRedaction(t *testing.T) {
	result, err := Render("ignored", Option{
		Sanitizer: func(any) (any, error) {
			return sanitizedValue{Token: "secret", Name: "visible"}, nil
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if result.JSON != `{"name":"visible","token":"<redacted>"}` || !result.Redacted {
		t.Fatalf("unexpected sanitized rendering: %#v", result)
	}
}

func TestRenderReturnsSanitizerErrors(t *testing.T) {
	cause := errors.New("sanitize failed")
	_, err := Render("value", Option{
		Sanitizer: func(any) (any, error) {
			return nil, cause
		},
	})
	if err == nil {
		t.Fatal("expected sanitizer error")
	}
	if failureKind(err) != "sanitize" || !errors.Is(err, cause) {
		t.Fatalf("unexpected sanitizer error: %v", err)
	}
}

func TestRenderSensitiveRootSkipsSanitizer(t *testing.T) {
	sanitizerCalls := 0
	result, err := Render("secret", Option{
		RootSensitive: true,
		Sanitizer: func(any) (any, error) {
			sanitizerCalls++
			return nil, errors.New("must not run")
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if sanitizerCalls != 0 || result.JSON != `"<redacted>"` || !result.Redacted {
		t.Fatalf("unexpected sensitive-root rendering: result=%#v calls=%d", result, sanitizerCalls)
	}
}

func TestRenderFailureLogsSafeDiagnostic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "redact-failure.jsonl")
	previousFailureLogger := failureLogger
	failureLogger = logger.New("vine:core:redact", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	})
	t.Cleanup(func() { failureLogger = previousFailureLogger })

	_, renderErr := Render("value", Option{
		Sanitizer: func(any) (any, error) {
			return nil, errors.New("secret must not be logged")
		},
	})
	if renderErr == nil {
		t.Fatal("expected render error")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failure log: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode failure log: %v", err)
	}
	if record["level"] != "ERROR" || record["logger"] != "vine:core:redact" ||
		record["msg"] != "value redaction failed" || record["failureKind"] != "sanitize" {
		t.Fatalf("unexpected failure log: %#v", record)
	}
	if strings.Contains(fmt.Sprint(record), "secret must not be logged") {
		t.Fatalf("failure log leaked the underlying error message: %#v", record)
	}
}

func TestDefaultLimitsArePositive(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxDepth <= 0 || limits.MaxNodes <= 0 || limits.MaxCollectionItems <= 0 ||
		limits.MaxStringBytes <= 0 || limits.MaxOutputBytes <= 0 {
		t.Fatalf("invalid default limits: %#v", limits)
	}
}

func TestRenderTruncatesFinalOutput(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxOutputBytes = 15
	result, err := Render(map[string]string{"name": "value"}, Option{Limits: &limits})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Truncated || result.Redacted ||
		!strings.Contains(result.JSON, "<truncated:json bytes=") ||
		strings.Contains(result.JSON, "sha256") {
		t.Fatalf("unexpected output truncation: %#v", result)
	}
}

func TestRenderRejectsInvalidLimits(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxNodes = 0
	if _, err := Render("value", Option{Limits: &limits}); err == nil {
		t.Fatal("expected invalid limits error")
	} else if failureKind(err) != "invalid_limits" {
		t.Fatalf("unexpected failure kind: %s", failureKind(err))
	}
}
