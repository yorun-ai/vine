package vstring

import (
	"fmt"
	"strings"

	"go.yorun.ai/vine/util/vpre"
)

// NeedsTrim reports whether trimming Unicode whitespace would change str.
func NeedsTrim(str string) bool {
	return strings.TrimSpace(str) != str
}

// IsBlank reports whether str is empty or contains only Unicode whitespace.
func IsBlank(str string) bool {
	return strings.TrimSpace(str) == ""
}

// EncodeDelimited encodes alternating name and value arguments as comma-separated name=value fields.
// It panics on an odd argument count, empty fields, commas, or equals signs.
func EncodeDelimited(pairs ...string) string {
	vpre.Check(len(pairs)%2 == 0, "delimited pairs must be even")

	parts := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		name := pairs[i]
		value := pairs[i+1]
		vpre.Check(name != "" && value != "", "delimited field cannot be empty")
		vpre.Check(!strings.ContainsAny(name, ",="), "delimited field name cannot contain comma or equals")
		vpre.Check(!strings.ContainsAny(value, ",="), "delimited field value cannot contain comma or equals")
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, ",")
}

// DecodeDelimited decodes comma-separated name=value fields.
// Field names must be unique and both names and values must be non-empty.
func DecodeDelimited(value string) (map[string]string, error) {
	fields := map[string]string{}
	for _, part := range strings.Split(value, ",") {
		name, fieldValue, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("missing value")
		}

		name = strings.TrimSpace(name)
		fieldValue = strings.TrimSpace(fieldValue)
		if name == "" || fieldValue == "" {
			return nil, fmt.Errorf("empty field")
		}

		if _, exists := fields[name]; exists {
			return nil, fmt.Errorf("duplicated field")
		}

		fields[name] = fieldValue
	}
	return fields, nil
}
