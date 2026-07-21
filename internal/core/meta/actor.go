package meta

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vpre"
)

// Actor

type ActorType string

const (
	ActorTypeAbsent         ActorType = "absent"
	ActorTypeAnonymous      ActorType = "anonymous"
	ActorTypeAuthenticating ActorType = "authenticating"
	ActorTypeAuthenticated  ActorType = "authenticated"
	ActorTypeImpersonated   ActorType = "impersonated"
)

type Actor interface {
	Type() ActorType
	IsAnonymous() bool
	IsAuthenticated() bool

	RawInfo() string

	mustByActor()
}

type _Actor struct {
	kind      ActorType
	actorInfo *_ActorInfo

	rawAuthInfo json.RawMessage
}

func (a *_Actor) Type() ActorType {
	return a.kind
}

func (a *_Actor) IsAnonymous() bool {
	return a.kind == ActorTypeAnonymous
}

func (a *_Actor) IsAuthenticated() bool {
	return a.kind == ActorTypeAuthenticated
}

func (a *_Actor) RawInfo() string {
	return string(a.rawAuthInfo)
}

func (a *_Actor) mustByActor() {}

func (a *_Actor) hasInfoType(infoType reflect.Type) bool {
	return infoType == a.actorInfo.InfoType
}

func NewAbsentActor() Actor {
	return &_Actor{kind: ActorTypeAbsent}
}

func NewAnonymousActor() Actor {
	return &_Actor{kind: ActorTypeAnonymous}
}

func NewAuthenticatingActor() Actor {
	return &_Actor{kind: ActorTypeAuthenticating}
}

func NewAuthenticatedActor[I any](info I) Actor {
	infoType := reflect.TypeFor[I]()
	actorInfo := infoByInfoType[infoType]
	vpre.CheckNotNil(actorInfo, "actor info type %s is not registered", infoType)

	rawAuthInfo, err := json.Marshal(info)
	vpre.CheckNilError(err, "marshal actor info %s failed", actorInfo.InfoSkelName)
	return &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   actorInfo,
		rawAuthInfo: rawAuthInfo,
	}
}

func NewAuthenticatedActorWithRawInfo(infoSkelName string, info json.RawMessage) Actor {
	return &_Actor{
		kind: ActorTypeAuthenticated,
		actorInfo: &_ActorInfo{
			InfoSkelName: infoSkelName,
		},
		rawAuthInfo: info,
	}
}

// Actor Info

func GetActorInfo[T any](metaActor Actor) (T, bool) {
	if info, ok := GetActorInfoByType(metaActor, reflect.TypeFor[T]()); ok {
		return info.(T), true
	}
	return *new(T), false
}

func MustGetActorInfo[T any](metaActor Actor) T {
	return MustGetActorInfoByType(metaActor, reflect.TypeFor[T]()).(T)
}

func GetActorInfoByType(metaActor Actor, kind reflect.Type) (any, bool) {
	actor := metaActor.(*_Actor)
	if actor.kind != ActorTypeAuthenticated {
		return nil, false
	}
	if !actor.hasInfoType(kind) {
		return nil, false
	}

	target := reflect.New(kind.Elem())
	err := json.Unmarshal(actor.rawAuthInfo, target.Interface())
	vpre.CheckNilError(err, "unmarshal actor info %s failed", actor.actorInfo.InfoSkelName)
	return target.Interface(), true
}

func MustGetActorInfoByType(metaActor Actor, kind reflect.Type) any {
	info, ok := GetActorInfoByType(metaActor, kind)
	vpre.Check(ok, "actor info type mismatch: %s", kind)
	return info
}

// Actor Payload

type _ActorPayload struct {
	Type         ActorType       `json:"type"`
	InfoSkelName string          `json:"infoSkelName,omitempty"`
	Info         json.RawMessage `json:"info,omitempty"`
}

func DecodeActorFromBase64(value string) (Actor, error) {
	payload, err := decodePayloadFromBase64[*_ActorPayload](value)
	if err != nil {
		return nil, err
	}

	switch payload.Type {
	case ActorTypeAbsent:
		return NewAbsentActor(), nil
	case ActorTypeAnonymous:
		return NewAnonymousActor(), nil
	case ActorTypeAuthenticating:
		return NewAuthenticatingActor(), nil
	case ActorTypeImpersonated:
		vpre.Panicf("impersonated actor not implemented")
	}

	if payload.InfoSkelName == "" || payload.Info == nil {
		return nil, fmt.Errorf("invalid payload %q", value)
	}

	actorInfo := infoByInfoSkelName[payload.InfoSkelName]
	vpre.Check(actorInfo != nil, "actor info %s is not registered", payload.InfoSkelName)

	return &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   actorInfo,
		rawAuthInfo: payload.Info,
	}, nil
}

func EncodeActorToBase64(metaActor Actor) string {
	actor := metaActor.(*_Actor)
	payload := &_ActorPayload{
		Type: actor.kind,
		Info: actor.rawAuthInfo,
	}
	if actor.actorInfo != nil {
		payload.InfoSkelName = actor.actorInfo.InfoSkelName
	}
	return encodePayloadToBase64(payload)
}
