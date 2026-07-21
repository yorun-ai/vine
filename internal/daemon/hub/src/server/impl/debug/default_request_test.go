package debug

import (
	"encoding/json"
	"testing"

	"go.yorun.ai/vine/internal/core/skel"
)

func TestDebugDefaultBuilderScalarValuesAreValidSkelJson(t *testing.T) {
	builder := _DebugDefaultBuilder{}

	assertValidSkelJsonValue[skel.UUID](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarUuid,
	}))
	assertValidSkelJsonValue[skel.Timestamp](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarTimestamp,
	}))
	assertValidSkelJsonValue[skel.Duration](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarDuration,
	}))
	assertValidSkelJsonValue[skel.LocalDate](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarLocalDate,
	}))
	assertValidSkelJsonValue[skel.LocalTime](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarLocalTime,
	}))
	assertValidSkelJsonValue[skel.LocalDateTime](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarLocalDateTime,
	}))
	assertValidSkelJsonValue[skel.Binary](t, builder.defaultValue(&skel.TypeSchema{
		Kind:   skel.TypeKindScalar,
		Scalar: skel.ScalarBinary,
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
