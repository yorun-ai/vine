package access

import (
	"testing"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/util/vcode"
)

func TestCborGetByPathSupportsFieldCascade(t *testing.T) {
	data := vcode.MustMarshalCbor(map[string]any{
		"params": map[string]any{
			"update": map[string]any{
				"userId": 42,
			},
		},
	})

	var payload any
	value, ok := cborGetByPath(&payload, data, "params.update.userId")
	if !ok {
		t.Fatalf("cborGet() ok = false, want true")
	}
	if value != uint64(42) {
		t.Fatalf("unexpected value: %#v", value)
	}
	if payload == nil {
		t.Fatalf("expected decoded payload to be cached")
	}
}

func TestCborGetByPathSupportsSingleWildcardPath(t *testing.T) {
	data := vcode.MustMarshalCbor(map[string]any{
		"params": map[string]any{
			"items": []any{
				map[string]any{"id": "first"},
				map[string]any{"id": "second"},
			},
		},
	})

	var payload any
	value, ok := cborGetByPath(&payload, data, "params.items[*].id")
	if !ok {
		t.Fatalf("cborGet() ok = false, want true")
	}
	values := value.([]any)
	if len(values) != 2 || values[0] != "first" || values[1] != "second" {
		t.Fatalf("unexpected value: %#v", value)
	}
}

func TestCborGetByPathRejectsUnsupportedWildcardPaths(t *testing.T) {
	data := vcode.MustMarshalCbor(map[string]any{
		"params": map[string]any{
			"items": []any{
				map[string]any{
					"children": []any{
						map[string]any{"id": "child"},
					},
				},
			},
		},
	})

	var payload any
	if _, ok := cborGetByPath(&payload, data, "params.items[*]"); ok {
		t.Fatalf("cborGet() ok = true for tail wildcard, want false")
	}
	if _, ok := cborGetByPath(&payload, data, "params.items[*].children[*].id"); ok {
		t.Fatalf("cborGet() ok = true for multiple wildcards, want false")
	}
}

func TestEvalPermExprKeepsAnyBranchesSeparate(t *testing.T) {
	ok, _, _, _ := evalPermExpr(&skeldesc.PermissionExpression{
		Mode: skeldesc.PermissionRequireModeAny,
		Children: []*skeldesc.PermissionExpression{
			{Mode: skeldesc.PermissionRequireModeCode, Code: "app.User:manage"},
			{
				Mode: skeldesc.PermissionRequireModeAll,
				Children: []*skeldesc.PermissionExpression{
					{Mode: skeldesc.PermissionRequireModeCode, Code: "app.User:update"},
				},
			},
		},
	}, map[string]bool{
		"app.User:manage": false,
		"app.User:update": true,
	}, func(*skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string) {
		t.Fatal("checkFunc should not be called")
		return false, ex.ServiceUnavailable, "", ""
	})
	if !ok {
		t.Fatalf("evalPermExpr() ok = false, want true")
	}
}

func TestEvalPermExprPreservesSelectedFailureReason(t *testing.T) {
	check := func(name string) *skeldesc.PermissionExpression {
		return &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeCheck, Check: &skeldesc.PermissionCheckInvocation{CheckName: name}}
	}
	for _, tc := range []struct {
		name        string
		expr        *skeldesc.PermissionExpression
		wantOK      bool
		wantMessage string
		wantReason  string
	}{
		{"all", &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeAll, Children: []*skeldesc.PermissionExpression{check("first"), check("last")}}, false, "first", "first_reason"},
		{"any", &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeAny, Children: []*skeldesc.PermissionExpression{check("first"), check("last")}}, false, "last", "last_reason"},
		{"any succeeds", &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeAny, Children: []*skeldesc.PermissionExpression{check("first"), {Mode: skeldesc.PermissionRequireModeCode, Code: "allowed"}}}, true, "", ""},
		{"any ends with code denial", &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeAny, Children: []*skeldesc.PermissionExpression{check("first"), {Mode: skeldesc.PermissionRequireModeCode, Code: "denied"}}}, false, "permission denied: denied", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, code, message, reason := evalPermExpr(tc.expr, map[string]bool{"allowed": true}, func(check *skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string) {
				return false, ex.PermissionDenied, check.CheckName, check.CheckName + "_reason"
			})
			wantCode := ex.PermissionDenied
			if tc.wantOK {
				wantCode = ex.OK
			}
			if ok != tc.wantOK || code != wantCode || message != tc.wantMessage || reason != tc.wantReason {
				t.Fatalf("got (%v, %s, %q, %q), want (%v, %s, %q, %q)", ok, code, message, reason, tc.wantOK, wantCode, tc.wantMessage, tc.wantReason)
			}
		})
	}
}
