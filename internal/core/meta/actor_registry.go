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

var infoBySkelName = map[string]*_ActorInfo{}
var infoByInfoSkelName = map[string]*_ActorInfo{}
var infoByInfoType = map[reflect.Type]*_ActorInfo{}

func RegisterActor(spec ActorSpec) {
	vpre.CheckNil(infoBySkelName[spec.SkelName], "actor %s already registered", spec.SkelName)

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
		infoByInfoSkelName[info.InfoSkelName] = info
		infoByInfoType[spec.InfoType] = info
	}

	infoBySkelName[info.SkelName] = info
}

func RegisteredActorInfoTypes() []reflect.Type {
	return vmap.Keys(infoByInfoType)
}
