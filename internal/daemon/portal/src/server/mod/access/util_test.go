package access

import (
	"testing"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel/legacy"
	"go.yorun.ai/vine/util/vcode"
	rpchttp "go.yorun.ai/vrpc/transport/http"
)

func TestEvalPermExprPreservesLegacyShortCircuitAfterConversion(t *testing.T) {
	for _, test := range []struct {
		mode    legacy.PermRequireMode
		allowed bool
		code    ex.Code
	}{
		{legacy.PermRequireModeAny, true, ex.OK},
		{legacy.PermRequireModeAll, false, ex.PermissionDenied},
	} {
		t.Run(string(test.mode), func(t *testing.T) {
			// Old generators expand Resource:action:check into all(code, check).
			declared := &legacy.PermExpr{
				Mode: test.mode,
				Children: []*legacy.PermExpr{
					{
						Mode: legacy.PermRequireModeAll,
						Children: []*legacy.PermExpr{
							{Mode: legacy.PermRequireModeCode, Code: "demo.Order:read"},
							{Mode: legacy.PermRequireModeCheck, Check: &legacy.PermCheckInvocation{
								ResourceSkelName: "demo.Order",
								ActionName:       "read",
								CheckName:        "exists",
							}},
						},
					},
					{Mode: legacy.PermRequireModeCode, Code: "demo.Order:update"},
				},
			}
			domain, err := legacy.Convert(&legacy.DomainSchema{
				Domain: "demo",
				Services: []*legacy.ServiceSchema{{
					Name:     "OrderApiService",
					Api:      true,
					AuthMode: legacy.AuthMode("required"),
					Methods: []*legacy.MethodSchema{{
						Name:    "read",
						Require: &legacy.PermRequire{Expr: declared},
					}},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := skeldesc.ValidateEffectivePolicy(domain); err != nil {
				t.Fatal(err)
			}
			calls := 0
			expression := domain.Services[0].Methods[0].EffectiveRequire.Expression
			allowed, code, _, _ := evalPermExpr(expression, map[string]bool{
				"demo.Order:read":   true,
				"demo.Order:update": test.allowed,
			}, func(*skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string) {
				calls++
				return false, ex.ServiceUnavailable, "resource service unavailable", ""
			})
			if allowed != test.allowed || code != test.code || calls != 0 {
				t.Fatalf("allowed=%v code=%s resource calls=%d; want allowed=%v code=%s without resource calls", allowed, code, calls, test.allowed, test.code)
			}
		})
	}
}

func TestRequestParamsGetByPathSupportsFieldCascade(t *testing.T) {
	data := vcode.MustMarshalCbor(map[string]any{
		"params": map[string]any{
			"update": map[string]any{
				"userId": 42,
			},
		},
	})

	var payload any
	value, ok := requestParamsGetByPath(&payload, data, rpchttp.ContentTypeCbor, "update.userId")
	if !ok {
		t.Fatalf("requestParamsGetByPath() ok = false, want true")
	}
	if value != uint64(42) {
		t.Fatalf("unexpected value: %#v", value)
	}
	if payload == nil {
		t.Fatalf("expected decoded payload to be cached")
	}
}

func TestRequestParamsGetByPathRejectsDuplicateKeysWithoutCachingPartialValue(t *testing.T) {
	for _, test := range []struct {
		contentType string
		body        []byte
	}{
		{
			contentType: rpchttp.ContentTypeJson,
			body:        []byte(`{"params":{"id":1,"id":2}}`),
		},
		{
			contentType: rpchttp.ContentTypeCbor,
			// {"params": {"id": 1, "id": 2}}
			body: []byte("\xa1\x66params\xa2\x62id\x01\x62id\x02"),
		},
	} {
		t.Run(test.contentType, func(t *testing.T) {
			var payload any
			value, ok := requestParamsGetByPath(&payload, test.body, test.contentType, "id")
			if ok || value != nil || payload != nil {
				t.Fatalf("ambiguous permission input accepted or cached: value=%v, cached=%v", value, payload)
			}
		})
	}
}

func TestRequestParamsGetByPathSupportsSingleWildcardPath(t *testing.T) {
	data := vcode.MustMarshalCbor(map[string]any{
		"params": map[string]any{
			"items": []any{
				map[string]any{"id": "first"},
				map[string]any{"id": "second"},
			},
		},
	})

	var payload any
	value, ok := requestParamsGetByPath(&payload, data, rpchttp.ContentTypeCbor, "items[*].id")
	if !ok {
		t.Fatalf("requestParamsGetByPath() ok = false, want true")
	}
	values := value.([]any)
	if len(values) != 2 || values[0] != "first" || values[1] != "second" {
		t.Fatalf("unexpected value: %#v", value)
	}
}

func TestRequestParamsGetByPathRejectsUnsupportedWildcardPaths(t *testing.T) {
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
	if _, ok := requestParamsGetByPath(&payload, data, rpchttp.ContentTypeCbor, "items[*]"); ok {
		t.Fatalf("requestParamsGetByPath() ok = true for tail wildcard, want false")
	}
	if _, ok := requestParamsGetByPath(&payload, data, rpchttp.ContentTypeCbor, "items[*].children[*].id"); ok {
		t.Fatalf("requestParamsGetByPath() ok = true for multiple wildcards, want false")
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
