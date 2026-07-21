package conf

import (
	"encoding/json"
	"reflect"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/util/vpre"
)

type Reader interface {
	GetByType(kind reflect.Type) any
}

type _Reader struct {
	linker link.Linker
}

func NewReader(linker link.Linker) Reader {
	return &_Reader{linker: linker}
}

func (r *_Reader) GetByType(kind reflect.Type) any {
	info := lookupByType(kind)
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
