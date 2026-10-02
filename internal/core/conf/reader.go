package conf

import (
	"encoding/json/v2"
	"reflect"

	"go.yorun.ai/vine/internal/core/link"
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
	return value.Interface()
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
