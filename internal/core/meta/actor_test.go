package meta

import (
	"reflect"
	"testing"
)

type _TestActorInfo struct {
	Name string
}

type _OtherTestActorInfo struct {
	Id string
}

func TestActorBase64RoundTripAnonymous(t *testing.T) {
	actor := NewAnonymousActor()

	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != actor.Type() {
		t.Fatalf("unexpected actor type: got=%s want=%s", got.Type(), actor.Type())
	}
	if got.RawInfo() != "" {
		t.Fatalf("unexpected actor info: %#v", got.RawInfo())
	}
}

func TestActorBase64RoundTripAuthenticating(t *testing.T) {
	actor := NewAuthenticatingActor()

	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != actor.Type() {
		t.Fatalf("unexpected actor type: got=%s want=%s", got.Type(), actor.Type())
	}
	if got.RawInfo() != "" {
		t.Fatalf("unexpected actor info: %#v", got.RawInfo())
	}
}

func TestDecodeActorFromEmptyBase64ReturnsError(t *testing.T) {
	_, err := DecodeActorFromBase64("")
	if err == nil {
		t.Fatalf("expected empty actor base64 error")
	}
}

func TestNewAnonymousActor(t *testing.T) {
	anonymous := NewAnonymousActor()
	if anonymous.Type() != ActorTypeAnonymous {
		t.Fatalf("unexpected anonymous actor type: %s", anonymous.Type())
	}
	if anonymous.RawInfo() != "" {
		t.Fatalf("unexpected anonymous actor info: %#v", anonymous.RawInfo())
	}
}

func TestNewAuthenticatingActor(t *testing.T) {
	authenticating := NewAuthenticatingActor()
	if authenticating.Type() != ActorTypeAuthenticating {
		t.Fatalf("unexpected authenticating actor type: %s", authenticating.Type())
	}
	if authenticating.RawInfo() != "" {
		t.Fatalf("unexpected authenticating actor info: %#v", authenticating.RawInfo())
	}
}

func TestNewAuthenticatedActor(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "NewAuthenticatedActor",
		SkelName:     "test.actor.NewAuthenticatedActor",
		InfoSkelName: "test.actor.NewAuthenticatedActorInfo",
		InfoType:     reflect.TypeFor[*_TestActorInfo](),
	}
	RegisterActor(spec)

	actor := NewAuthenticatedActor(&_TestActorInfo{Name: "demo"})

	if actor.Type() != ActorTypeAuthenticated {
		t.Fatalf("unexpected actor type: %s", actor.Type())
	}
	if actor.RawInfo() != `{"Name":"demo"}` {
		t.Fatalf("unexpected actor info: %s", actor.RawInfo())
	}
	info := MustGetActorInfo[*_TestActorInfo](actor)
	if info.Name != "demo" {
		t.Fatalf("unexpected actor info: %#v", info)
	}
	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if MustGetActorInfo[*_TestActorInfo](got).Name != "demo" {
		t.Fatalf("unexpected round trip actor info: %#v", got)
	}
}

func TestNewAuthenticatedActorRejectsUnregisteredInfoType(t *testing.T) {
	resetActorRegistryForTest()

	assertPanics(t, func() {
		NewAuthenticatedActor(&_TestActorInfo{Name: "demo"})
	})
}

func TestNewAbsentActor(t *testing.T) {
	absentActor := NewAbsentActor()

	if absentActor.Type() != ActorTypeAbsent {
		t.Fatalf("unexpected absent actor type: %s", absentActor.Type())
	}
	if absentActor.RawInfo() != "" {
		t.Fatalf("unexpected absent actor info: %#v", absentActor.RawInfo())
	}
}

func TestActorTypePredicates(t *testing.T) {
	tests := []struct {
		name          string
		actor         Actor
		anonymous     bool
		authenticated bool
	}{
		{
			name:  "absent",
			actor: NewAbsentActor(),
		},
		{
			name:      "anonymous",
			actor:     NewAnonymousActor(),
			anonymous: true,
		},
		{
			name:  "authenticating",
			actor: NewAuthenticatingActor(),
		},
		{
			name:          "authenticated",
			actor:         NewAuthenticatedActorWithRawInfo("test.actor.Info", []byte(`{"Name":"demo"}`)),
			authenticated: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.actor.IsAnonymous() != tc.anonymous {
				t.Fatalf("IsAnonymous() = %t, want %t", tc.actor.IsAnonymous(), tc.anonymous)
			}
			if tc.actor.IsAuthenticated() != tc.authenticated {
				t.Fatalf("IsAuthenticated() = %t, want %t", tc.actor.IsAuthenticated(), tc.authenticated)
			}
		})
	}
}

func TestActorBase64RoundTripWithInfo(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "Base64InfoActor",
		SkelName:     "test.actor.Base64InfoActor",
		InfoSkelName: "test.actor.Base64InfoActorInfo",
		InfoType:     reflect.TypeFor[*_TestActorInfo](),
	}
	RegisterActor(spec)

	actor := &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   infoByInfoType[spec.InfoType],
		rawAuthInfo: []byte(`{"Name":"demo"}`),
	}
	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != ActorTypeAuthenticated {
		t.Fatalf("unexpected actor type: %s", got.Type())
	}
	if got.RawInfo() != `{"Name":"demo"}` {
		t.Fatalf("unexpected actor info: %s", got.RawInfo())
	}
}

func TestNewAuthorizedActorBase64RoundTrip(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "AuthorizedActor",
		SkelName:     "test.actor.AuthorizedActor",
		InfoSkelName: "test.actor.AuthorizedActorInfo",
		InfoType:     reflect.TypeFor[*_TestActorInfo](),
	}
	RegisterActor(spec)

	actor := NewAuthenticatedActorWithRawInfo(spec.InfoSkelName, []byte(`{"Name":"demo"}`))
	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != ActorTypeAuthenticated {
		t.Fatalf("unexpected actor type: %s", got.Type())
	}
	if got.RawInfo() != `{"Name":"demo"}` {
		t.Fatalf("unexpected actor info: %s", got.RawInfo())
	}
}

func TestGetActorInfoByType(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "TryInfoActor",
		SkelName:     "test.actor.TryInfoActor",
		InfoSkelName: "test.actor.TryInfoActorInfo",
		InfoType:     reflect.TypeFor[*_TestActorInfo](),
	}
	RegisterActor(spec)

	actor, err := DecodeActorFromBase64(EncodeActorToBase64(NewAuthenticatedActorWithRawInfo(spec.InfoSkelName, []byte(`{"Name":"demo"}`))))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}

	info, ok := GetActorInfoByType(actor, reflect.TypeFor[*_TestActorInfo]())
	if !ok {
		t.Fatalf("expected actor info")
	}
	if info.(*_TestActorInfo).Name != "demo" {
		t.Fatalf("unexpected actor info: %#v", info)
	}

	_, ok = GetActorInfoByType(actor, reflect.TypeFor[*_OtherTestActorInfo]())
	if ok {
		t.Fatalf("expected actor info mismatch")
	}
}

func TestActorBase64RoundTripWithImpersonatedTypePanics(t *testing.T) {
	actor := &_Actor{kind: ActorTypeImpersonated}

	assertPanics(t, func() {
		_, _ = DecodeActorFromBase64(EncodeActorToBase64(actor))
	})
}

func TestActorBase64RoundTripWithAbsentType(t *testing.T) {
	actor := NewAbsentActor()

	got, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	if got.Type() != actor.Type() {
		t.Fatalf("unexpected actor type: got=%s want=%s", got.Type(), actor.Type())
	}
}
