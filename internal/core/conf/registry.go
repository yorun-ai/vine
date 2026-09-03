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

type Registry struct {
	infoBySkelName map[string]*_ConfigInfo
	infoByType     map[reflect.Type]*_ConfigInfo
}

func NewRegistry() *Registry {
	return &Registry{
		infoBySkelName: map[string]*_ConfigInfo{},
		infoByType:     map[reflect.Type]*_ConfigInfo{},
	}
}

var defaultRegistry = NewRegistry()

func Register(spec ConfigSpec) {
	defaultRegistry.Register(spec)
}

func (r *Registry) Register(spec ConfigSpec) {
	vpre.CheckNil(r.infoBySkelName[spec.SkelName], "config %s already registered", spec.SkelName)

	info := &_ConfigInfo{
		Name:      spec.Name,
		SkelName:  spec.SkelName,
		Hash:      spec.Hash,
		Lifecycle: spec.Lifecycle,
		Type:      spec.Type,
	}
	r.infoBySkelName[info.SkelName] = info
	r.infoByType[spec.Type] = info
}

func RegisteredTypes() []reflect.Type {
	return defaultRegistry.RegisteredTypes()
}

func (r *Registry) RegisteredTypes() []reflect.Type {
	return vmap.Keys(r.infoByType)
}

func SkelNameByType(kind reflect.Type) string {
	return defaultRegistry.SkelNameByType(kind)
}

func (r *Registry) SkelNameByType(kind reflect.Type) string {
	return r.lookupByType(kind).SkelName
}

func (r *Registry) lookupByType(kind reflect.Type) *_ConfigInfo {
	info := r.infoByType[kind]
	vpre.CheckNotNil(info, "config type %s is not registered", kind)
	return info
}
