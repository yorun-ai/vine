package meta

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
	"log/slog"
	"reflect"

	"go.yorun.ai/vine/util/vcode"
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
	// IsImpersonated reports whether an operator is acting as this identity.
	IsImpersonated() bool
	// Impersonator returns the operator, or nil for ordinary authentication.
	Impersonator() Actor

	Realm() string
	Identifier() string

	RawInfo() string

	// LogFields returns identity fields without actor info payloads.
	LogFields() []slog.Attr

	mustByActor()
}

type _Actor struct {
	kind      ActorType
	actorInfo *_ActorInfo

	realm      string
	identifier string

	rawAuthInfo  jsontext.Value
	impersonator *_Actor
}

func (a *_Actor) Type() ActorType {
	return a.kind
}

func (a *_Actor) IsAnonymous() bool {
	return a.kind == ActorTypeAnonymous
}

func (a *_Actor) IsAuthenticated() bool {
	return a.kind == ActorTypeAuthenticated || a.kind == ActorTypeImpersonated
}

func (a *_Actor) IsImpersonated() bool {
	return a.kind == ActorTypeImpersonated
}

func (a *_Actor) Impersonator() Actor {
	if a.impersonator == nil {
		return nil
	}
	return a.impersonator
}

// NewImpersonatedActor preserves the effective identity and attaches its operator.
func NewImpersonatedActor(actor Actor, impersonator Actor) Actor {
	vpre.Check(actor != nil && actor.IsAuthenticated(), "impersonation requires an authenticated effective actor")
	vpre.Check(impersonator != nil && impersonator.IsAuthenticated(), "impersonation requires an authenticated operator")
	result := *actor.(*_Actor)
	result.kind = ActorTypeImpersonated
	result.impersonator = impersonator.(*_Actor)
	return &result
}

func (a *_Actor) Realm() string {
	return a.realm
}

func (a *_Actor) Identifier() string {
	return a.identifier
}

func (a *_Actor) RawInfo() string {
	return string(a.rawAuthInfo)
}

// LogFields describes the effective identity and its direct impersonator.
// Empty identity values and actor info payloads are omitted.
func (a *_Actor) LogFields() []slog.Attr {
	fields := make([]slog.Attr, 0, 5)
	fields = append(fields, slog.String("actorType", string(a.Type())))
	if realm := a.Realm(); realm != "" {
		fields = append(fields, slog.String("actorRealm", realm))
	}
	if identifier := a.Identifier(); identifier != "" {
		fields = append(fields, slog.String("actorIdentifier", identifier))
	}
	if impersonator := a.Impersonator(); impersonator != nil {
		if realm := impersonator.Realm(); realm != "" {
			fields = append(fields, slog.String("impersonatorActorRealm", realm))
		}
		if identifier := impersonator.Identifier(); identifier != "" {
			fields = append(fields, slog.String("impersonatorActorIdentifier", identifier))
		}
	}
	return fields
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
	actorInfo := defaultRegistry.infoByInfoType[infoType]
	vpre.CheckNotNil(actorInfo, "actor info type %s is not registered", infoType)

	rawAuthInfo, err := vcode.MarshalJson(info)
	vpre.CheckNilError(err, "marshal actor info %s failed", actorInfo.InfoSkelName)
	return &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   actorInfo,
		realm:       actorInfo.SkelName,
		identifier:  actorInfo.identifier(info),
		rawAuthInfo: rawAuthInfo,
	}
}

func NewAuthenticatedActorWithRawInfo(realm string, identifier string, infoSkelName string, info jsontext.Value) Actor {
	return &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   &_ActorInfo{InfoSkelName: infoSkelName},
		realm:       realm,
		identifier:  identifier,
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
	if metaActor == nil {
		return nil, false
	}
	actor := metaActor.(*_Actor)
	if !actor.IsAuthenticated() {
		return nil, false
	}
	if !actor.hasInfoType(kind) {
		return nil, false
	}

	target := reflect.New(kind.Elem())
	err := vcode.UnmarshalJsonTo(actor.rawAuthInfo, target.Interface())
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
	Type ActorType `json:"type"`

	Realm      string `json:"realm,omitempty"`
	Identifier string `json:"identifier,omitempty"`

	InfoSkelName string         `json:"infoSkelName,omitempty"`
	Info         jsontext.Value `json:"info,omitempty"`
	Impersonator *_ActorPayload `json:"impersonator,omitempty"`
}

func DecodeActorFromBase64(value string) (Actor, error) {
	payload, err := decodePayloadFromBase64[*_ActorPayload](value)
	if err != nil {
		return nil, err
	}

	return decodeActorPayload(payload, 0)
}

func decodeActorPayload(payload *_ActorPayload, depth int) (Actor, error) {
	if payload == nil || depth > 32 {
		return nil, fmt.Errorf("invalid actor payload")
	}
	if payload.Type != ActorTypeImpersonated && payload.Impersonator != nil {
		return nil, fmt.Errorf("unexpected impersonator for actor type %s", payload.Type)
	}
	switch payload.Type {
	case ActorTypeAbsent:
		return NewAbsentActor(), nil
	case ActorTypeAnonymous:
		return NewAnonymousActor(), nil
	case ActorTypeAuthenticating:
		return NewAuthenticatingActor(), nil
	case ActorTypeAuthenticated, ActorTypeImpersonated:
	default:
		return nil, fmt.Errorf("unknown actor type %s", payload.Type)
	}
	payload.Info = bytes.TrimSpace(payload.Info)
	if payload.InfoSkelName == "" || !payload.Info.IsValid() || len(payload.Info) == 0 || payload.Info[0] != '{' {
		return nil, fmt.Errorf("invalid actor info payload")
	}
	actorInfo := defaultRegistry.infoByInfoSkelName[payload.InfoSkelName]
	if actorInfo == nil {
		// Transit processes and consumers need not import every operator's Go type.
		actorInfo = new(_ActorInfo{InfoSkelName: payload.InfoSkelName})
	}
	actor := new(_Actor{
		kind:        payload.Type,
		actorInfo:   actorInfo,
		realm:       payload.Realm,
		identifier:  payload.Identifier,
		rawAuthInfo: payload.Info,
	})
	if payload.Type == ActorTypeImpersonated {
		operator, err := decodeActorPayload(payload.Impersonator, depth+1)
		if err != nil {
			return nil, err
		}
		if !operator.IsAuthenticated() {
			return nil, fmt.Errorf("impersonator must be authenticated")
		}
		actor.impersonator = operator.(*_Actor)
	}
	return actor, nil
}

func EncodeActorToBase64(metaActor Actor) string {
	return encodePayloadToBase64(actorPayload(metaActor.(*_Actor)))
}

func actorPayload(actor *_Actor) *_ActorPayload {
	payload := new(_ActorPayload{
		Type:       actor.kind,
		Realm:      actor.Realm(),
		Identifier: actor.Identifier(),
		Info:       actor.rawAuthInfo,
	})
	if actor.actorInfo != nil {
		payload.InfoSkelName = actor.actorInfo.InfoSkelName
	}
	if actor.impersonator != nil {
		payload.Impersonator = actorPayload(actor.impersonator)
	}
	return payload
}
