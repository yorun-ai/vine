package access

import (
	"slices"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/tidwall/gjson"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
)

// Supported JsonPath syntax:
//   - field cascade, such as "params.update.userId"
//   - at most one list wildcard in a non-tail segment, such as
//     "params.users[*].id"
//
// Unsupported path syntax includes tail wildcards like "params.users[*]",
// multiple wildcards, array indexes, filters, slices, recursive descent, and
// quoted fields. Tail wildcards are rejected because "items[*]" has the same
// permission-check meaning as "items" and only adds ambiguity.

func jsonGetByPath(data []byte, jsonPath string) (any, bool) {
	if _, ok := parseJsonPath(jsonPath); !ok {
		return nil, false
	}
	gjsonPath := strings.ReplaceAll(jsonPath, "[*]", ".#")
	value := gjson.GetBytes(data, gjsonPath)
	return value.Value(), value.Exists()
}

func cborGetByPath(payload *any, data []byte, jsonPath string) (any, bool) {
	parts, ok := parseJsonPath(jsonPath)
	if !ok {
		return nil, false
	}

	if *payload != nil {
		return cborGetPathPartsValue(*payload, parts)
	}

	var value any
	if err := cbor.Unmarshal(data, &value); err != nil {
		return nil, false
	}

	*payload = value
	return cborGetPathPartsValue(value, parts)
}

type _JsonPathPart struct {
	name     string
	wildcard bool
}

func parseJsonPath(path string) ([]_JsonPathPart, bool) {
	rawParts := strings.Split(path, ".")
	parts := make([]_JsonPathPart, 0, len(rawParts))
	wildcardCount := 0
	for _, rawPart := range rawParts {
		if rawPart == "" {
			return nil, false
		}
		part := _JsonPathPart{
			name: rawPart,
		}
		if before, ok := strings.CutSuffix(rawPart, "[*]"); ok {
			part.name = before
			part.wildcard = true
			wildcardCount++
		}
		if part.name == "" || strings.ContainsAny(part.name, "[]") || wildcardCount > 1 {
			return nil, false
		}
		parts = append(parts, part)
	}
	if parts[len(parts)-1].wildcard {
		return nil, false
	}
	return parts, true
}

func cborGetPathPartsValue(value any, parts []_JsonPathPart) (any, bool) {
	for index, part := range parts {
		var ok bool
		value, ok = cborSelectPathField(value, part.name)
		if !ok {
			return nil, false
		}
		if part.wildcard {
			return cborSelectWildcardPathValues(value, parts[index+1:])
		}
	}
	return value, true
}

func cborSelectPathField(value any, part string) (any, bool) {
	switch node := value.(type) {
	case map[string]any:
		value, ok := node[part]
		return value, ok
	case map[any]any:
		value, ok := node[part]
		return value, ok
	default:
		return nil, false
	}
}

func cborSelectWildcardPathValues(value any, remainingParts []_JsonPathPart) (any, bool) {
	values, ok := cborAsAnySlice(value)
	if !ok {
		return nil, false
	}
	results := make([]any, 0, len(values))
	for _, item := range values {
		value, ok := cborGetPathPartsValue(item, remainingParts)
		if !ok {
			return nil, false
		}
		results = append(results, value)
	}
	return results, true
}

func cborAsAnySlice(value any) ([]any, bool) {
	switch values := value.(type) {
	case []any:
		return values, true
	default:
		return nil, false
	}
}

func collectPermissionCodes(expr *skeldesc.PermissionExpression) []string {
	codes := make([]string, 0)
	seen := map[string]struct{}{}
	collectPermissionCodesTo(expr, seen, &codes)
	return codes
}

func collectPermissionCodesTo(expr *skeldesc.PermissionExpression, seen map[string]struct{}, codes *[]string) {
	if expr.Mode == skeldesc.PermissionRequireModeCode {
		if _, ok := seen[expr.Code]; !ok {
			seen[expr.Code] = struct{}{}
			*codes = append(*codes, expr.Code)
		}
		return
	}

	for _, child := range expr.Children {
		collectPermissionCodesTo(child, seen, codes)
	}
}

func hasPermissionChecks(expr *skeldesc.PermissionExpression) bool {
	if expr.Mode == skeldesc.PermissionRequireModeCheck {
		return true
	}

	return slices.ContainsFunc(expr.Children, hasPermissionChecks)
}

func evalPermExpr(expr *skeldesc.PermissionExpression, codeResults map[string]bool, checkFunc func(*skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string)) (bool, ex.Code, string, string) {
	switch expr.Mode {
	case skeldesc.PermissionRequireModeCode:
		if codeResults[expr.Code] {
			return true, ex.OK, "", ""
		}
		return false, ex.PermissionDenied, "permission denied: " + expr.Code, ""
	case skeldesc.PermissionRequireModeCheck:
		return checkFunc(expr.Check)
	case skeldesc.PermissionRequireModeAll:
		for _, child := range expr.Children {
			ok, code, message, reason := evalPermExpr(child, codeResults, checkFunc)
			if !ok {
				return false, code, message, reason
			}
		}
		return true, ex.OK, "", ""
	case skeldesc.PermissionRequireModeAny:
		return evalAnyPermExpr(expr.Children, codeResults, checkFunc)
	default:
		return false, ex.ServiceUnavailable, "unsupported permission require mode", ""
	}
}

func evalAnyPermExpr(children []*skeldesc.PermissionExpression, codeResults map[string]bool, checkFunc func(*skeldesc.PermissionCheckInvocation) (bool, ex.Code, string, string)) (bool, ex.Code, string, string) {
	code := ex.ClientForbidden
	message := "permission check failed"
	reason := ""
	for _, child := range children {
		ok, checkCode, checkMessage, checkReason := evalPermExpr(child, codeResults, checkFunc)
		if ok {
			return true, ex.OK, "", ""
		}
		code = checkCode
		message = checkMessage
		reason = checkReason
	}
	return false, code, message, reason
}
