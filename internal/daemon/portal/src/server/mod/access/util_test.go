package access

import (
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
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
	ok, _, _, _ := evalPermExpr(&skel.PermExpr{
		Mode: skel.PermRequireModeAny,
		Children: []*skel.PermExpr{
			{Mode: skel.PermRequireModeCode, Code: "app.User:manage"},
			{
				Mode: skel.PermRequireModeAll,
				Children: []*skel.PermExpr{
					{Mode: skel.PermRequireModeCode, Code: "app.User:update"},
				},
			},
		},
	}, map[string]bool{
		"app.User:manage": false,
		"app.User:update": true,
	}, func(*skel.PermCheckInvocation) (bool, ex.Code, string, string) {
		t.Fatal("checkFunc should not be called")
		return false, ex.ServiceUnavailable, "", ""
	})
	if !ok {
		t.Fatalf("evalPermExpr() ok = false, want true")
	}
}

func TestReorderPermExprDelaysChecksInSameGroup(t *testing.T) {
	expr := reorderPermExpr(&skel.PermExpr{
		Mode: skel.PermRequireModeAny,
		Children: []*skel.PermExpr{
			{Mode: skel.PermRequireModeCheck, Check: &skel.PermCheckInvocation{CheckName: "byOwner"}},
			{
				Mode: skel.PermRequireModeAll,
				Children: []*skel.PermExpr{
					{Mode: skel.PermRequireModeCheck, Check: &skel.PermCheckInvocation{CheckName: "byTenant"}},
					{Mode: skel.PermRequireModeCode, Code: "app.User:update"},
				},
			},
			{Mode: skel.PermRequireModeCode, Code: "app.User:manage"},
		},
	})

	if expr.Children[0].Mode != skel.PermRequireModeCode || expr.Children[0].Code != "app.User:manage" {
		t.Fatalf("first child = %#v, want manage code", expr.Children[0])
	}
	if expr.Children[1].Mode != skel.PermRequireModeAll {
		t.Fatalf("second child mode = %s, want all", expr.Children[1].Mode)
	}
	if expr.Children[2].Mode != skel.PermRequireModeCheck {
		t.Fatalf("third child mode = %s, want check", expr.Children[2].Mode)
	}
	if expr.Children[1].Children[0].Mode != skel.PermRequireModeCode {
		t.Fatalf("nested first child mode = %s, want code", expr.Children[1].Children[0].Mode)
	}

	ok, _, _, _ := evalPermExpr(expr, map[string]bool{
		"app.User:manage": true,
		"app.User:update": true,
	}, func(*skel.PermCheckInvocation) (bool, ex.Code, string, string) {
		t.Fatal("checkFunc should not be called after code branch succeeds")
		return false, ex.ServiceUnavailable, "", ""
	})
	if !ok {
		t.Fatalf("evalPermExpr() ok = false, want true")
	}
}

func TestEvalPermExprPreservesSelectedFailureReason(t *testing.T) {
	check := func(name string) *skel.PermExpr {
		return &skel.PermExpr{Mode: skel.PermRequireModeCheck, Check: &skel.PermCheckInvocation{CheckName: name}}
	}
	for _, tc := range []struct {
		name        string
		expr        *skel.PermExpr
		wantOK      bool
		wantMessage string
		wantReason  string
	}{
		{"all", &skel.PermExpr{Mode: skel.PermRequireModeAll, Children: []*skel.PermExpr{check("first"), check("last")}}, false, "first", "first_reason"},
		{"any", &skel.PermExpr{Mode: skel.PermRequireModeAny, Children: []*skel.PermExpr{check("first"), check("last")}}, false, "last", "last_reason"},
		{"any succeeds", &skel.PermExpr{Mode: skel.PermRequireModeAny, Children: []*skel.PermExpr{check("first"), {Mode: skel.PermRequireModeCode, Code: "allowed"}}}, true, "", ""},
		{"any ends with code denial", &skel.PermExpr{Mode: skel.PermRequireModeAny, Children: []*skel.PermExpr{check("first"), {Mode: skel.PermRequireModeCode, Code: "denied"}}}, false, "permission denied: denied", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, code, message, reason := evalPermExpr(tc.expr, map[string]bool{"allowed": true}, func(check *skel.PermCheckInvocation) (bool, ex.Code, string, string) {
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
