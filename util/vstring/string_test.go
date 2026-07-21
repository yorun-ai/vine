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
