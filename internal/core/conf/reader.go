package conf

import (
	"encoding/json/v2"
	"reflect"
	"strings"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vpre"
)

type Reader interface {
	GetByType(kind reflect.Type) any
}

type _Reader struct {
	linker   link.Linker
	registry *Registry
}

func NewReader(linker link.Linker) Reader {
	return newReader(linker, defaultRegistry)
}

func newReader(linker link.Linker, registry *Registry) Reader {
	return &_Reader{linker: linker, registry: registry}
}

func (r *_Reader) GetByType(kind reflect.Type) any {
	info := r.registry.lookupByType(kind)
	text := r.getRaw(info)
	vpre.Check(text != "", "config %s json is empty", info.SkelName)

	value := reflect.New(kind.Elem())
	err := json.Unmarshal([]byte(text), value.Interface())
	vpre.CheckNilError(err, "unmarshal config %s failed", info.SkelName)
	for info, field := range value.Elem().Fields() {
		if field.CanSet() && !skel.HasTagFlag(info.Tag, "noTrim") {
			trimConfigStrings(field)
		}
	}
	return value.Interface()
}

// Only Go strings represent Skel string fields. Named string types, including
// skel.JSON and enums, retain their own semantics and must not be trimmed.
func trimConfigStrings(value reflect.Value) {
	if value.Type() == reflect.TypeFor[string]() {
		value.SetString(strings.TrimSpace(value.String()))
		return
	}

	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			trimConfigStrings(value.Elem())
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			trimConfigStrings(value.Index(i))
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			entry := reflect.New(value.Type().Elem()).Elem()
			entry.Set(iter.Value())
			trimConfigStrings(entry)
			value.SetMapIndex(iter.Key(), entry)
		}
	default:
		// Other kinds, including named strings and scalar structs, stay unchanged.
		return
	}
}

func (r *_Reader) getRaw(info *_ConfigInfo) string {
	switch info.Lifecycle {
	case LifecycleEternal:
		return r.linker.ConfigClient().GetEternal(info.SkelName)
	case LifecycleInstant:
		return r.linker.ConfigClient().GetInstant(info.SkelName)
	default:
		return ""
	}
}
