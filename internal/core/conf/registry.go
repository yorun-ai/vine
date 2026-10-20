package conf

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

// Spec

// ConfigSpec describes a configuration definition and its optional abstract source.
type ConfigSpec struct {
	Name      string
	SkelName  string
	Hash      string
	Lifecycle Lifecycle
	Type      reflect.Type
	// Abs marks a contract that cannot be resolved or assigned an application value.
	Abs bool
	// FromSkelName names the abstract contract adopted by this concrete config.
	FromSkelName string
	// FromType describes the source contract pointer type.
	// Source metadata is emitted by Skel and is not validated at runtime.
	FromType reflect.Type
}

// Info

type _ConfigInfo struct {
	Name         string
	SkelName     string
	Hash         string
	Lifecycle    Lifecycle
	Type         reflect.Type
	Abs          bool
	FromSkelName string
	FromType     reflect.Type
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

	vpre.CheckNil(r.infoByType[spec.Type], "config type %s already registered", spec.Type)
	info := &_ConfigInfo{
		Name:         spec.Name,
		SkelName:     spec.SkelName,
		Hash:         spec.Hash,
		Lifecycle:    spec.Lifecycle,
		Type:         spec.Type,
		Abs:          spec.Abs,
		FromSkelName: spec.FromSkelName,
		FromType:     spec.FromType,
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

// RequireConcreteType rejects abstract configs.
func RequireConcreteType(kind reflect.Type) {
	defaultRegistry.requireConcreteType(kind)
}

func (r *Registry) requireConcreteType(kind reflect.Type) *_ConfigInfo {
	info := r.lookupByType(kind)
	vpre.Check(!info.Abs, "abstract config %s cannot be resolved or assigned a value", info.SkelName)
	return info
}

// ValidateValueName rejects values for known abstract contracts. Unknown names
// remain valid so test seeds may precede application contract registration.
func ValidateValueName(name string) error { return defaultRegistry.ValidateValueName(name) }

// ValidateValueName checks a configuration value name against this registry.
func (r *Registry) ValidateValueName(name string) error {
	if info := r.infoBySkelName[name]; info != nil && info.Abs {
		return fmt.Errorf("abstract config %s cannot be assigned a value", name)
	}
	return nil
}
