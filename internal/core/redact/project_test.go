package redact

import (
	"strings"
	"testing"
)

type taggedValue struct {
	Name   string `json:"name"`
	Secret string `json:"customSecret" skel:"sensitive"`
}

func TestRenderMasksOnlyExplicitlySensitiveFields(t *testing.T) {
	result, err := Render(struct {
		Profile taggedValue       `json:"profile"`
		Meta    map[string]string `json:"meta"`
	}{
		Profile: taggedValue{Name: "Alice", Secret: "tagged-secret"},
		Meta:    map[string]string{"access_token": "key-secret", "visible": "value"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Redacted {
		t.Fatal("expected redacted result")
	}
	if strings.Contains(result.JSON, "tagged-secret") {
		t.Fatalf("sensitive value leaked: %s", result.JSON)
	}
	if !strings.Contains(result.JSON, `"customSecret":"<redacted>"`) ||
		!strings.Contains(result.JSON, `"access_token":"key-secret"`) {
		t.Fatalf("unexpected rendering: %s", result.JSON)
	}
}

func TestRenderHandlesCycles(t *testing.T) {
	value := map[string]any{}
	value["self"] = value
	result, err := Render(value)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(result.JSON, "<cycle>") {
		t.Fatalf("unexpected cycle rendering: %s", result.JSON)
	}
}

func TestRenderTruncatesLongStringsWithoutDigest(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxStringBytes = 5
	result, err := Render("123456", Option{Limits: &limits})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if result.JSON != `"<truncated:string bytes=6>"` || !result.Truncated || result.Redacted {
		t.Fatalf("unexpected truncated string: %#v", result)
	}
	if strings.Contains(result.JSON, "sha256") {
		t.Fatalf("truncation must not calculate a digest: %s", result.JSON)
	}
}

func TestRenderTruncatesOversizedCollections(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxCollectionItems = 2
	for name, value := range map[string]any{
		"list": []int{1, 2, 3},
		"map":  map[string]int{"a": 1, "b": 2, "c": 3},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := Render(value, Option{Limits: &limits})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !result.Truncated || result.Redacted || !strings.Contains(result.JSON, "<truncated:"+name) {
				t.Fatalf("unexpected truncated collection: %#v", result)
			}
		})
	}
}

func TestRenderTruncatesAtDepthAndNodeLimits(t *testing.T) {
	depthLimits := DefaultLimits()
	depthLimits.MaxDepth = 1
	depthResult, err := Render(map[string]any{
		"child": map[string]any{"value": "hidden"},
	}, Option{Limits: &depthLimits})
	if err != nil {
		t.Fatalf("render depth: %v", err)
	}
	if !depthResult.Truncated || !strings.Contains(depthResult.JSON, "<truncated:depth limit=1>") {
		t.Fatalf("unexpected depth result: %#v", depthResult)
	}

	nodeLimits := DefaultLimits()
	nodeLimits.MaxNodes = 2
	nodeResult, err := Render(map[string]string{
		"a": "first",
		"b": "second",
	}, Option{Limits: &nodeLimits})
	if err != nil {
		t.Fatalf("render nodes: %v", err)
	}
	if !nodeResult.Truncated || !strings.Contains(nodeResult.JSON, "<truncated:nodes limit=2>") {
		t.Fatalf("unexpected node result: %#v", nodeResult)
	}
}

func TestRenderLimitsStillApplyWhenSensitiveValuesAreRevealed(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxStringBytes = 3
	result, err := Render("visible", Option{
		RevealSensitive: true,
		Limits:          &limits,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !result.Truncated || result.JSON != `"<truncated:string bytes=7>"` {
		t.Fatalf("unexpected revealed truncation: %#v", result)
	}
}
