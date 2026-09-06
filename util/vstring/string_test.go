package vstring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsTrim(t *testing.T) {
	assert.False(t, NeedsTrim(""))
	assert.False(t, NeedsTrim("demo"))
	assert.False(t, NeedsTrim("go.yorun.ai/app/vine"))
	assert.True(t, NeedsTrim(" demo"))
	assert.True(t, NeedsTrim("demo "))
	assert.True(t, NeedsTrim("\tdemo\n"))
}

func TestIsBlank(t *testing.T) {
	assert.True(t, IsBlank(""))
	assert.True(t, IsBlank("   "))
	assert.True(t, IsBlank("\t\n"))
	assert.False(t, IsBlank("demo"))
	assert.False(t, IsBlank(" demo "))
}

func TestTrimSpacePtr(t *testing.T) {
	assert.Equal(t, "", TrimSpacePtr(nil))
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"", ""},
		{" \t\n\u2003\u00a0", ""},
		{"demo", "demo"},
		{"\u2003 hello  world\ninside \u00a0", "hello  world\ninside"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			value := tt.input
			assert.Equal(t, tt.want, TrimSpacePtr(&value))
			assert.Equal(t, tt.input, value, "must not mutate the input")
		})
	}
}

func TestFirstNonBlank(t *testing.T) {
	assert.Equal(t, "", FirstNonBlank())
	assert.Equal(t, "", FirstNonBlank("", " \t\n", "\u2003\u00a0"))
	values := []string{"", "\u2003", "\u00a0 first  value\ninside \t", "second"}
	original := append([]string(nil), values...)
	assert.Equal(t, "first  value\ninside", FirstNonBlank(values...))
	assert.Equal(t, original, values, "must not mutate the inputs")
	assert.Equal(t, "first", FirstNonBlank("first", "second"))
}

func TestEncodeDelimitedKeepsPairOrder(t *testing.T) {
	got := EncodeDelimited(
		"name", "demo.app",
		"version", "1.0.0",
		"instanceId", "123e4567-e89b-12d3-a456-426614174000",
	)

	assert.Equal(t, "name=demo.app,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000", got)
}

func TestEncodeDelimitedRejectsInvalidPairs(t *testing.T) {
	assert.Panics(t, func() { EncodeDelimited("name") })
	assert.Panics(t, func() { EncodeDelimited("", "demo") })
	assert.Panics(t, func() { EncodeDelimited("name", "") })
	assert.Panics(t, func() { EncodeDelimited("na=me", "demo") })
	assert.Panics(t, func() { EncodeDelimited("name", "de,mo") })
}

func TestDecodeDelimited(t *testing.T) {
	got, err := DecodeDelimited("name=demo.app, version=1.0.0, instanceId=123e4567-e89b-12d3-a456-426614174000")

	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"name":       "demo.app",
		"version":    "1.0.0",
		"instanceId": "123e4567-e89b-12d3-a456-426614174000",
	}, got)
}

func TestDecodeDelimitedRejectsInvalidValue(t *testing.T) {
	_, err := DecodeDelimited("name")
	assert.Error(t, err)

	_, err = DecodeDelimited("name=")
	assert.Error(t, err)

	_, err = DecodeDelimited("name=demo,name=other")
	assert.Error(t, err)
}
