package spec

import (
	"fmt"
	"strconv"

	"go.yorun.ai/vine/internal/util/reflectutil"
)

func CheckValueNotNil(value any, path string) error {
	if reflectutil.IsNil(value) {
		return fmt.Errorf("%s cannot be nil", path)
	}
	return nil
}

func JoinPath(base string, field string) string {
	if base == "" {
		return field
	}
	return base + "." + field
}

func JoinIndex(base string, index int) string {
	return fmt.Sprintf("%s[%d]", base, index)
}

func JoinMapKey(base string, key any) string {
	switch v := key.(type) {
	case string:
		return base + "[" + strconv.Quote(v) + "]"
	case fmt.Stringer:
		return base + "[" + strconv.Quote(v.String()) + "]"
	default:
		return fmt.Sprintf("%s[%v]", base, key)
	}
}
