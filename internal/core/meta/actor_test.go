package meta

import (
	"reflect"
	"testing"
	"uuid"
)

type _TestActorInfo struct {
	Name string
}

type _OtherTestActorInfo struct {
	Id string
}

func TestActorBase64RoundTripNonAuthenticated(t *testing.T) {
	tests := []struct {
		name  string
		kind  ActorType
		actor Actor
	}{
		{name: "absent", kind: ActorTypeAbsent, actor: NewAbsentActor()},
		{name: "anonymous", kind: ActorTypeAnonymous, actor: NewAnonymousActor()},
		{name: "authenticating", kind: ActorTypeAuthenticating, actor: NewAuthenticatingActor()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeActorFromBase64(EncodeActorToBase64(test.actor))
			if err != nil {
				t.Fatalf("DecodeActorFromBase64() error = %v", err)
			}
			if got.Type() != test.kind {
				t.Fatalf("unexpected actor type: got=%s want=%s", got.Type(), test.kind)
			}
			if got.RawInfo() != "" {
				t.Fatalf("unexpected actor info: %#v", got.RawInfo())
			}
		})
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
			actor:         NewAuthenticatedActorWithRawInfo("test.actor.Actor", "", "test.actor.Info", []byte(`{"Name":"demo"}`)),
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

func TestNewAuthorizedActorBase64RoundTrip(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "AuthorizedActor",
		SkelName:     "test.actor.AuthorizedActor",
		InfoSkelName: "test.actor.AuthorizedActorInfo",
		InfoType:     reflect.TypeFor[*_TestActorInfo](),
	}
	RegisterActor(spec)

	actor := NewAuthenticatedActorWithRawInfo(spec.SkelName, "", spec.InfoSkelName, []byte(`{"Name":"demo"}`))
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

	actor, err := DecodeActorFromBase64(EncodeActorToBase64(NewAuthenticatedActorWithRawInfo(spec.SkelName, "", spec.InfoSkelName, []byte(`{"Name":"demo"}`))))
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

func TestActorIdentityRoundTrip(t *testing.T) {
	previousRegistry := defaultRegistry
	defaultRegistry = NewRegistry()
	t.Cleanup(func() { defaultRegistry = previousRegistry })
	type identityInfo struct {
		ID int64 `json:"id" skel:"identifier"`
	}
	spec := ActorSpec{SkelName: "base.UserActor", InfoSkelName: "base.UserActorInfo", InfoType: reflect.TypeFor[*identityInfo]()}
	RegisterActor(spec)
	actor := NewAuthenticatedActor(new(identityInfo{ID: 9007199254740993}))
	raw := NewAuthenticatedActorWithRawInfo(spec.SkelName, "9007199254740993", spec.InfoSkelName, []byte(`{"id":9007199254740993}`))
	for _, candidate := range []Actor{actor, raw} {
		got, err := DecodeActorFromBase64(EncodeActorToBase64(candidate))
		if err != nil {
			t.Fatal(err)
		}
		if got.Realm() != spec.SkelName || got.Identifier() != "9007199254740993" {
			t.Fatalf("wrong identity: %s/%s", got.Realm(), got.Identifier())
		}
		if MustGetActorInfo[*identityInfo](got).ID != 9007199254740993 {
			t.Fatal("info lost precision")
		}
	}
	forwarded, err := DecodeActorFromBase64(encodePayloadToBase64(&_ActorPayload{
		Type: ActorTypeAuthenticated, Realm: "forwarded.UserActor", Identifier: "forwarded-id",
		InfoSkelName: spec.InfoSkelName, Info: []byte(`{"id":1}`),
	}))
	if err != nil || forwarded.Realm() != "forwarded.UserActor" || forwarded.Identifier() != "forwarded-id" {
		t.Fatalf("forwarded identity changed: %v", err)
	}
	for _, candidate := range []Actor{NewAbsentActor(), NewAnonymousActor(), NewAuthenticatingActor()} {
		if candidate.Realm() != "" || candidate.Identifier() != "" {
			t.Fatal("unexpected unauthenticated identity")
		}
	}
}

func TestActorIdentifierTag(t *testing.T) {
	previousRegistry := defaultRegistry
	defaultRegistry = NewRegistry()
	t.Cleanup(func() { defaultRegistry = previousRegistry })
	type stringInfo struct {
		Name string `json:"name"`
		ID   string `json:"id" skel:"sensitive,identifier"`
	}
	type uuidInfo struct {
		ID uuid.UUID `json:"id" skel:"identifier"`
	}
	RegisterActor(ActorSpec{SkelName: "base.StringActor", InfoSkelName: "base.StringInfo", InfoType: reflect.TypeFor[*stringInfo]()})
	RegisterActor(ActorSpec{SkelName: "base.UUIDActor", InfoSkelName: "base.UUIDInfo", InfoType: reflect.TypeFor[*uuidInfo]()})
	if got := NewAuthenticatedActor(new(stringInfo{Name: "other", ID: "user-1"})).Identifier(); got != "user-1" {
		t.Fatalf("unexpected string identifier: %s", got)
	}
	id := uuid.MustParse("12345678-1234-5678-9abc-123456789abc")
	if got := NewAuthenticatedActor(new(uuidInfo{ID: id})).Identifier(); got != id.String() {
		t.Fatalf("unexpected UUID identifier: %s", got)
	}
}
