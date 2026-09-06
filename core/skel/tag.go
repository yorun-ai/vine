package skel

import (
	"reflect"

	internalskel "go.yorun.ai/vine/internal/core/skel"
)

// HasTagFlag reports whether the comma-separated skel struct tag contains name
// as an exact flag, ignoring whitespace around each attribute.
func HasTagFlag(tag reflect.StructTag, name string) bool {
	return internalskel.HasTagFlag(tag, name)
}

// TagIndex reads index(n) from the skel struct tag. It reports malformed or
// repeated indexes as errors and ignores unrelated attributes. It does not read
// legacy arg tags or validate the index against an argument structure's size.
func TagIndex(tag reflect.StructTag) (index int, found bool, err error) {
	return internalskel.TagIndex(tag)
}
