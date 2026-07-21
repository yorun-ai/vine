package access

import (
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/tidwall/gjson"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/skel"
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
		part := _JsonPathPart{name: rawPart}
		if strings.HasSuffix(rawPart, "[*]") {
			part.name = strings.TrimSuffix(rawPart, "[*]")
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
	case map[interface{}]interface{}:
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

// Permission expressions keep the schema shape but are reordered for runtime
// execution. Inside each direct all()/any() child list, cheap code checks are
// evaluated first, nested expressions stay in the middle, and RPC-backed check
// calls are delayed to the end. Reordering is local to the current expression
// level; nested all()/any() groups are recursively reordered but never flattened
// into their parent.
//
// Permission code results are collected from the full expression before
// evaluation and every requested code must be present in the actor permission
// service response. Resource check calls stay lazy: evalPermExpr only invokes
// checkFunc when the reordered short-circuit traversal reaches a check node.

func mergeRequirements(requirements []*skel.PermRequire) *skel.PermExpr {
	if len(requirements) == 1 {
		return requirements[0].Expr
	}

	children := make([]*skel.PermExpr, 0, len(requirements))
	for _, require := range requirements {
		children = append(children, require.Expr)
	}
	return &skel.PermExpr{
		Mode:     skel.PermRequireModeAll,
		Children: children,
	}
}

func reorderPermExpr(expr *skel.PermExpr) *skel.PermExpr {
	if expr.Mode != skel.PermRequireModeAll && expr.Mode != skel.PermRequireModeAny {
		return expr
	}

	children := make([]*skel.PermExpr, 0, len(expr.Children))
	for _, child := range expr.Children {
		children = append(children, reorderPermExpr(child))
	}

	return &skel.PermExpr{
		Mode:     expr.Mode,
		Children: reorderPermExprChildren(children),
	}
}

func reorderPermExprChildren(children []*skel.PermExpr) []*skel.PermExpr {
	reordered := make([]*skel.PermExpr, 0, len(children))
	for rank := 0; rank <= 2; rank++ {
		for _, child := range children {
			if permExprRank(child) == rank {
				reordered = append(reordered, child)
			}
		}
	}
	return reordered
}

func permExprRank(expr *skel.PermExpr) int {
	switch expr.Mode {
	case skel.PermRequireModeCode:
		return 0
	case skel.PermRequireModeCheck:
		return 2
	default:
		return 1
	}
}

func collectPermissionCodes(expr *skel.PermExpr) []string {
	codes := make([]string, 0)
	seen := map[string]struct{}{}
	collectPermissionCodesTo(expr, seen, &codes)
	return codes
}

func collectPermissionCodesTo(expr *skel.PermExpr, seen map[string]struct{}, codes *[]string) {
	if expr.Mode == skel.PermRequireModeCode {
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

func hasPermissionChecks(expr *skel.PermExpr) bool {
	if expr.Mode == skel.PermRequireModeCheck {
		return true
	}

	for _, child := range expr.Children {
		if hasPermissionChecks(child) {
			return true
		}
	}
	return false
}

func evalPermExpr(expr *skel.PermExpr, codeResults map[string]bool, checkFunc func(*skel.PermCheckInvocation) (bool, ex.Code, string)) (bool, ex.Code, string) {
	switch expr.Mode {
	case skel.PermRequireModeCode:
		if codeResults[expr.Code] {
			return true, ex.OK, ""
		}
		return false, ex.PermissionDenied, "permission denied: " + expr.Code
	case skel.PermRequireModeCheck:
		return checkFunc(expr.Check)
	case skel.PermRequireModeAll:
		for _, child := range expr.Children {
			ok, code, message := evalPermExpr(child, codeResults, checkFunc)
			if !ok {
				return false, code, message
			}
		}
		return true, ex.OK, ""
	case skel.PermRequireModeAny:
		return evalAnyPermExpr(expr.Children, codeResults, checkFunc)
	default:
		return false, ex.ServiceUnavailable, "unsupported permission require mode"
	}
}

func evalAnyPermExpr(children []*skel.PermExpr, codeResults map[string]bool, checkFunc func(*skel.PermCheckInvocation) (bool, ex.Code, string)) (bool, ex.Code, string) {
	code := ex.ClientForbidden
	message := "permission check failed"
	for _, child := range children {
		ok, checkCode, checkMessage := evalPermExpr(child, codeResults, checkFunc)
		if ok {
			return true, ex.OK, ""
		}
		code = checkCode
		message = checkMessage
	}
	return false, code, message
}
