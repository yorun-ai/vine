package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type FieldDirectTagged struct {
	Primary *RequestScope `inject:""`
	Ignored *RequestScope
}

type FieldEmbeddedDependency struct {
	Dependency *RequestScope `inject:""`
}

type FieldEmbeddedPointerDependency struct {
	Dependency *SeededRequestContext `inject:""`
}

type FieldAnonymousInterface interface {
	Do()
}

type fieldScanSkipsUnsupportedAnonymous struct {
	FieldAnonymousInterface
	FieldEmbeddedDependency
}

type fieldScanEmbeddedPointerStruct struct {
	*FieldEmbeddedPointerDependency
}

type fieldScanEmbeddedPointerStructWithoutInjectedFields struct {
	*scopeNoMarker
}

type fieldScanNestedEmbedded struct {
	FieldEmbeddedDependency
	FieldDirectTagged
}

type fieldTaggedUnexported struct {
	hidden *RequestScope `inject:""`
}

type fieldTaggedAnonymous struct {
	FieldEmbeddedDependency `inject:""`
}

func TestScanInjectedFieldsReadsDirectTaggedFields(t *testing.T) {
	fields := scanInjectedFields(T[*FieldDirectTagged]())

	if assert.Len(t, fields, 1) {
		assert.Equal(t, "Primary", fields[0].name)
		assert.Equal(t, []int{0}, fields[0].index)
		assert.Equal(t, T[*RequestScope](), fields[0].kind)
	}
}

func TestScanInjectedFieldsSkipsUnsupportedAnonymousFieldTypes(t *testing.T) {
	fields := scanInjectedFields(T[*fieldScanSkipsUnsupportedAnonymous]())

	if assert.Len(t, fields, 1) {
		assert.Equal(t, "FieldEmbeddedDependency.Dependency", fields[0].name)
		assert.Equal(t, []int{1, 0}, fields[0].index)
		assert.Equal(t, T[*RequestScope](), fields[0].kind)
	}
}

func TestScanInjectedFieldsAllowsAnonymousPointerStructWithoutInjectedFields(t *testing.T) {
	fields := scanInjectedFields(T[*fieldScanEmbeddedPointerStructWithoutInjectedFields]())

	assert.Empty(t, fields)
}

func TestScanInjectedFieldsPanicsOnAnonymousPointerStructWithInjectedFields(t *testing.T) {
	assert.PanicsWithError(t,
		"embedded pointer struct FieldEmbeddedPointerDependency of fieldScanEmbeddedPointerStruct contains injected fields, use value embedding instead",
		func() {
			_ = scanInjectedFields(T[*fieldScanEmbeddedPointerStruct]())
		},
	)
}

func TestScanInjectedFieldsCollectsNestedEmbeddedAndDirectFields(t *testing.T) {
	fields := scanInjectedFields(T[*fieldScanNestedEmbedded]())

	if assert.Len(t, fields, 2) {
		assert.Equal(t, "FieldEmbeddedDependency.Dependency", fields[0].name)
		assert.Equal(t, []int{0, 0}, fields[0].index)
		assert.Equal(t, T[*RequestScope](), fields[0].kind)

		assert.Equal(t, "FieldDirectTagged.Primary", fields[1].name)
		assert.Equal(t, []int{1, 0}, fields[1].index)
		assert.Equal(t, T[*RequestScope](), fields[1].kind)
	}
}

func TestScanInjectedFieldsPanicsOnTaggedUnexportedField(t *testing.T) {
	assert.Panics(t, func() {
		_ = scanInjectedFields(T[*fieldTaggedUnexported]())
	})
}

func TestScanInjectedFieldsPanicsOnTaggedAnonymousField(t *testing.T) {
	assert.Panics(t, func() {
		_ = scanInjectedFields(T[*fieldTaggedAnonymous]())
	})
}
