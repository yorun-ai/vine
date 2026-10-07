package debug

import (
	"encoding/json/v2"
	"testing"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
)

func TestDebugDefaultBuilderScalarValuesAreValidSkelJson(t *testing.T) {
	builder := _DebugDefaultBuilder{}

	assertValidSkelJsonValue[skeltype.UUID](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarUUID,
	}))
	assertValidSkelJsonValue[skeltype.Timestamp](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarTimestamp,
	}))
	assertValidSkelJsonValue[skeltype.Duration](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarDuration,
	}))
	assertValidSkelJsonValue[skeltype.LocalDate](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarLocalDate,
	}))
	assertValidSkelJsonValue[skeltype.LocalTime](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarLocalTime,
	}))
	assertValidSkelJsonValue[skeltype.LocalDateTime](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarLocalDateTime,
	}))
	assertValidSkelJsonValue[skeltype.Binary](t, builder.defaultValue(&skeldesc.Type{
		Kind:   skeldesc.TypeKindScalar,
		Scalar: skeldesc.ScalarBinary,
	}))
}

func assertValidSkelJsonValue[T any](t *testing.T, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal(%#v) error = %v", value, err)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", string(data), err)
	}
}
