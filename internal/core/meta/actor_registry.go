package meta

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

// Spec

type ActorSpec struct {
	Name         string
	SkelName     string
	Hash         string
	InfoSkelName string
	InfoType     reflect.Type
}

// Info

type _ActorInfo struct {
	Name         string
	SkelName     string
	Hash         string
	InfoSkelName string
	InfoType     reflect.Type
}

// Registry

type Registry struct {
	infoBySkelName     map[string]*_ActorInfo
	infoByInfoSkelName map[string]*_ActorInfo
	infoByInfoType     map[reflect.Type]*_ActorInfo
}

func NewRegistry() *Registry {
	return &Registry{
		infoBySkelName:     map[string]*_ActorInfo{},
		infoByInfoSkelName: map[string]*_ActorInfo{},
		infoByInfoType:     map[reflect.Type]*_ActorInfo{},
	}
}

var defaultRegistry = NewRegistry()

func RegisterActor(spec ActorSpec) {
	defaultRegistry.RegisterActor(spec)
}

func (r *Registry) RegisterActor(spec ActorSpec) {
	vpre.CheckNil(r.infoBySkelName[spec.SkelName], "actor %s already registered", spec.SkelName)

	info := &_ActorInfo{
		Name:         spec.Name,
		SkelName:     spec.SkelName,
		Hash:         spec.Hash,
		InfoSkelName: spec.InfoSkelName,
		InfoType:     spec.InfoType,
	}

	if spec.InfoType != nil {
		vpre.Check(reflectutil.IsStructPointerType(spec.InfoType),
			"actor %s info type %s must be pointer to struct", spec.SkelName, spec.InfoType)
		r.infoByInfoSkelName[info.InfoSkelName] = info
		r.infoByInfoType[spec.InfoType] = info
	}

	r.infoBySkelName[info.SkelName] = info
}

func RegisteredActorInfoTypes() []reflect.Type {
	return defaultRegistry.RegisteredActorInfoTypes()
}

func (r *Registry) RegisteredActorInfoTypes() []reflect.Type {
	return vmap.Keys(r.infoByInfoType)
}
