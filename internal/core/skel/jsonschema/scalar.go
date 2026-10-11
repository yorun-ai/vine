package jsonschema

import (
	"fmt"

	"go.yorun.ai/skel/descriptor"
)

func scalar(kind descriptor.Scalar) (Schema, error) {
	result := Schema{"x-skel-type": string(kind)}
	switch kind {
	case descriptor.ScalarString:
		result["type"] = "string"
	case descriptor.ScalarBoolean:
		result["type"] = "boolean"
	case descriptor.ScalarInt:
		result["type"] = "integer"
	case descriptor.ScalarFloat:
		result["type"] = "number"
	case descriptor.ScalarJSON:
		// Arbitrary JSON includes null, even without a nullable modifier.
	case descriptor.ScalarDecimal:
		result["type"] = "string"
		result["pattern"] = `^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`
	case descriptor.ScalarBinary:
		result["type"] = "string"
		result["contentEncoding"] = "base64"
	case descriptor.ScalarUUID:
		result["type"] = "string"
		result["format"] = "uuid"
	case descriptor.ScalarTimestamp:
		result["type"] = "string"
		result["format"] = "date-time"
	case descriptor.ScalarLocalDate:
		result["type"] = "string"
		result["format"] = "date"
	case descriptor.ScalarDuration:
		// Skel uses time.ParseDuration, not JSON Schema's ISO 8601 duration.
		result["type"] = "string"
	case descriptor.ScalarLocalTime:
		result["type"] = "string"
		result["pattern"] = `^(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](?:\.[0-9]+)?$`
	case descriptor.ScalarLocalDateTime:
		result["type"] = "string"
		result["pattern"] = `^[0-9]{4}-[0-9]{2}-[0-9]{2}T(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](?:\.[0-9]+)?$`
	default:
		return nil, fmt.Errorf("unsupported scalar %q", kind)
	}
	return result, nil
}
