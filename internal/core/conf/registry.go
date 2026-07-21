package conf

import (
	"reflect"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

// Spec

type ConfigSpec struct {
	Name      string
	SkelName  string
	Hash      string
	Lifecycle Lifecycle
	Type      reflect.Type
}

// Info

type _ConfigInfo struct {
	Name      string
	SkelName  string
	Hash      string
	Lifecycle Lifecycle
	Type      reflect.Type
}

// Registry

var infoBySkelName = map[string]*_ConfigInfo{}
var infoByType = map[reflect.Type]*_ConfigInfo{}

func Register(spec ConfigSpec) {
	vpre.CheckNil(infoBySkelName[spec.SkelName], "config %s already registered", spec.SkelName)

	info := &_ConfigInfo{
		Name:      spec.Name,
		SkelName:  spec.SkelName,
		Hash:      spec.Hash,
		Lifecycle: spec.Lifecycle,
		Type:      spec.Type,
	}
	infoBySkelName[info.SkelName] = info
	infoByType[spec.Type] = info
}

func RegisteredTypes() []reflect.Type {
	return vmap.Keys(infoByType)
}

func SkelNameByType(kind reflect.Type) string {
	return lookupByType(kind).SkelName
}

func lookupByType(kind reflect.Type) *_ConfigInfo {
	info := infoByType[kind]
	vpre.CheckNotNil(info, "config type %s is not registered", kind)
	return info
}
