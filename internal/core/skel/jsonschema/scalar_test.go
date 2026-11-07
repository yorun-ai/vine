package jsonschema

import (
	"encoding/json/v2"
	"regexp"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
)

func TestScalarsDescribeSharedJSONWireValues(t *testing.T) {
	for _, test := range []struct {
		kind     descriptor.Scalar
		value    any
		jsonType string
		format   string
	}{
		{descriptor.ScalarString, "hi", "string", ""},
		{descriptor.ScalarBoolean, true, "boolean", ""},
		{descriptor.ScalarInt, 42, "integer", ""},
		{descriptor.ScalarFloat, 1.5, "number", ""},
		{descriptor.ScalarDecimal, skeltype.NewDecimal(decimal.RequireFromString("1.500")), "string", ""},
		{descriptor.ScalarBinary, skeltype.Binary{1, 2, 3}, "string", ""},
		{descriptor.ScalarTimestamp, skeltype.NewTimestamp(time.Now()), "string", "date-time"},
		{descriptor.ScalarDuration, skeltype.NewDuration(time.Hour), "string", ""},
		{descriptor.ScalarLocalDate, skeltype.NewLocalDateOf(time.Now()), "string", "date"},
		{descriptor.ScalarLocalTime, skeltype.NewLocalTimeOf(time.Now()), "string", ""},
		{descriptor.ScalarLocalDateTime, skeltype.NewLocalDateTimeOf(time.Now()), "string", ""},
		{descriptor.ScalarUUID, "123e4567-e89b-12d3-a456-426614174000", "string", "uuid"},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			schema, err := scalar(test.kind)
			require.NoError(t, err)
			require.Equal(t, test.jsonType, schema["type"])
			if test.format == "" {
				require.NotContains(t, schema, "format")
			} else {
				require.Equal(t, test.format, schema["format"])
			}
			wire, err := json.Marshal(test.value)
			require.NoError(t, err)
			if test.jsonType == "string" {
				var value string
				require.NoError(t, json.Unmarshal(wire, &value))
				if pattern, ok := schema["pattern"]; ok {
					require.Regexp(t, regexp.MustCompile(pattern.(string)), value)
				}
			}
		})
	}
	json, err := scalar(descriptor.ScalarJSON)
	require.NoError(t, err)
	require.NotContains(t, json, "type", "arbitrary JSON includes null")
	binary, err := scalar(descriptor.ScalarBinary)
	require.NoError(t, err)
	require.Equal(t, "base64", binary["contentEncoding"])
}
