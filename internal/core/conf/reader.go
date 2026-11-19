package conf

import (
	"encoding/json/v2"
	"reflect"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type Reader interface {
	GetByType(kind reflect.Type) any
}

type _Reader struct {
	linker   link.Linker
	registry *Registry
	eternal  *vmap.MutexMap[string, string]
}

func NewReader(linker link.Linker) Reader {
	return newReader(linker, defaultRegistry)
}

func newReader(linker link.Linker, registry *Registry) Reader {
	return &_Reader{linker: linker, registry: registry, eternal: vmap.NewMutexMap[string, string]()}
}

func (r *_Reader) GetByType(kind reflect.Type) any {
	info := r.registry.requireConcreteType(kind)
	var raw string
	switch info.Lifecycle {
	case LifecycleInstant:
		raw = r.linker.ConfigClient().GetInstant(info.SkelName)
	case LifecycleEternal:
		return r.getEternal(info)
	}
	return decodeConfig(info, raw)
}

func (r *_Reader) getEternal(info *_ConfigInfo) any {
	// The app can read configuration before Link knows its instance. Keep the
	// first successfully decoded value for the lifetime of this app's reader.
	if raw, ok := r.eternal.Load(info.SkelName); ok {
		return decodeConfig(info, raw)
	}
	raw := r.linker.ConfigClient().GetEternal(info.SkelName)
	value := decodeConfig(info, raw)
	if first, loaded := r.eternal.LoadOrStore(info.SkelName, raw); loaded && first != raw {
		return decodeConfig(info, first)
	}
	return value
}

func decodeConfig(info *_ConfigInfo, raw string) any {
	vpre.Check(raw != "", "config %s json is empty", info.SkelName)
	value := reflect.New(info.Type.Elem())
	err := json.Unmarshal([]byte(raw), value.Interface())
	vpre.CheckNilError(err, "unmarshal config %s failed", info.SkelName)
	return value.Interface()
}
