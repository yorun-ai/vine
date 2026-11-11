package meta

import (
	"reflect"
	"strings"
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

func TestGetActorInfoWithoutAuthenticatedActor(t *testing.T) {
	for name, actor := range map[string]Actor{
		"nil":            nil,
		"absent":         NewAbsentActor(),
		"anonymous":      NewAnonymousActor(),
		"authenticating": NewAuthenticatingActor(),
	} {
		t.Run(name, func(t *testing.T) {
			info, ok := GetActorInfoByType(actor, reflect.TypeFor[*_TestActorInfo]())
			if ok || info != nil {
				t.Fatalf("expected no actor info, got %#v, %v", info, ok)
			}
			typedInfo, ok := GetActorInfo[*_TestActorInfo](actor)
			if ok || typedInfo != nil {
				t.Fatalf("expected no typed actor info, got %#v, %v", typedInfo, ok)
			}
		})
	}
}

func TestImpersonatedActorRoundTrip(t *testing.T) {
	previous := defaultRegistry
	defaultRegistry = NewRegistry()
	t.Cleanup(func() { defaultRegistry = previous })
	RegisterActor(ActorSpec{SkelName: "demo.User", InfoSkelName: "demo.UserInfo", InfoType: reflect.TypeFor[*_TestActorInfo]()})
	effective := NewAuthenticatedActor(new(_TestActorInfo{Name: "user"}))
	operator := NewAuthenticatedActorWithRawInfo("ops.Operator", "9007199254740993", "ops.OperatorInfo", []byte(`{"id":9007199254740993,"scope":"all"}`))
	actor := NewImpersonatedActor(effective, operator)
	if effective.IsImpersonated() || effective.Impersonator() != nil {
		t.Fatal("constructor mutated effective actor")
	}
	for range 2 {
		decoded, err := DecodeActorFromBase64(EncodeActorToBase64(actor))
		if err != nil {
			t.Fatal(err)
		}
		if !decoded.IsAuthenticated() || !decoded.IsImpersonated() || decoded.Realm() != effective.Realm() {
			t.Fatal("effective identity changed")
		}
		info, ok := GetActorInfo[*_TestActorInfo](decoded)
		if !ok || info.Name != "user" {
			t.Fatal("typed effective info unavailable")
		}
		op := decoded.Impersonator()
		if op.Identifier() != "9007199254740993" || op.RawInfo() != operator.RawInfo() || op.Realm() != operator.Realm() {
			t.Fatal("operator lost")
		}
		if op.Type() != ActorTypeAuthenticated || op.Impersonator() != nil {
			t.Fatal("operator must be ordinarily authenticated")
		}
		if _, ok := GetActorInfo[*_TestActorInfo](op); ok {
			t.Fatal("unregistered operator confused with effective actor")
		}
		actor = decoded
	}
}

func TestInvalidActorPayloadReturnsError(t *testing.T) {
	for _, payload := range []*_ActorPayload{
		nil,
		{Type: "unknown"},
		{Type: ActorTypeImpersonated, InfoSkelName: "demo.Info", Info: []byte(`{}`)},
		{Type: ActorTypeImpersonated, InfoSkelName: "demo.Info", Info: []byte(`{}`), Impersonator: new(_ActorPayload{Type: ActorTypeAnonymous})},
		{Type: ActorTypeAuthenticated, InfoSkelName: "demo.Info", Info: []byte(`null`)},
		{Type: ActorTypeAuthenticated, InfoSkelName: "demo.Info", Info: []byte(`[]`)},
		{Type: ActorTypeAnonymous, Impersonator: new(_ActorPayload{Type: ActorTypeAnonymous})},
	} {
		if _, err := DecodeActorFromBase64(encodePayloadToBase64(payload)); err == nil {
			t.Fatalf("accepted invalid payload: %#v", payload)
		}
	}
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
		Type:         ActorTypeAuthenticated,
		Realm:        "forwarded.UserActor",
		Identifier:   "forwarded-id",
		InfoSkelName: spec.InfoSkelName,
		Info:         []byte(`{"id":1}`),
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

func TestActorLogFields(t *testing.T) {
	user := NewAuthenticatedActorWithRawInfo("demo.UserActor", "9007199254740993", "demo.UserActorInfo", []byte(`{"secret":"not-for-logs"}`))
	operator := NewAuthenticatedActorWithRawInfo("demo.OpsActor", "ops-1", "demo.OpsActorInfo", []byte(`{"secret":"operator-secret"}`))
	withoutID := NewAuthenticatedActorWithRawInfo("demo.OpsActor", "", "demo.OpsActorInfo", []byte(`{}`))
	for _, tt := range []struct {
		name  string
		actor Actor
		want  map[string]any
	}{
		{name: "absent", actor: NewAbsentActor(), want: map[string]any{"actorType": "absent"}},
		{name: "anonymous", actor: NewAnonymousActor(), want: map[string]any{"actorType": "anonymous"}},
		{name: "authenticating", actor: NewAuthenticatingActor(), want: map[string]any{"actorType": "authenticating"}},
		{name: "authenticated", actor: user, want: map[string]any{
			"actorType":       "authenticated",
			"actorRealm":      "demo.UserActor",
			"actorIdentifier": "9007199254740993",
		}},
		{name: "no identifier", actor: withoutID, want: map[string]any{
			"actorType":  "authenticated",
			"actorRealm": "demo.OpsActor",
		}},
		{name: "impersonated", actor: NewImpersonatedActor(user, operator), want: map[string]any{
			"actorType":                   "impersonated",
			"actorRealm":                  "demo.UserActor",
			"actorIdentifier":             "9007199254740993",
			"impersonatorActorRealm":      "demo.OpsActor",
			"impersonatorActorIdentifier": "ops-1",
		}},
		{name: "operator without identifier", actor: NewImpersonatedActor(user, withoutID), want: map[string]any{
			"actorType":              "impersonated",
			"actorRealm":             "demo.UserActor",
			"actorIdentifier":        "9007199254740993",
			"impersonatorActorRealm": "demo.OpsActor",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := map[string]any{}
			for _, attr := range tt.actor.LogFields() {
				got[attr.Key] = attr.Value.Any()
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("log fields = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewImpersonatedActorRejectsInvalidIdentities(t *testing.T) {
	user := NewAuthenticatedActorWithRawInfo("demo.UserActor", "user", "demo.UserActorInfo", []byte(`{"id":"user"}`))
	operator := NewAuthenticatedActorWithRawInfo("demo.OpsActor", "ops", "demo.OpsActorInfo", []byte(`{"id":"ops"}`))
	impersonated := NewImpersonatedActor(user, operator)
	for _, tc := range []struct {
		name  string
		actor Actor
	}{
		{name: "nil"},
		{name: "absent", actor: NewAbsentActor()},
		{name: "anonymous", actor: NewAnonymousActor()},
		{name: "authenticating", actor: NewAuthenticatingActor()},
		{name: "already impersonated", actor: impersonated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() { NewImpersonatedActor(tc.actor, operator) })
			assertPanics(t, func() { NewImpersonatedActor(user, tc.actor) })
		})
	}
	if user.IsImpersonated() || operator.IsImpersonated() || impersonated.Impersonator() != operator {
		t.Fatal("rejected construction mutated an existing identity")
	}
}

func TestDecodeActorRejectsNestedImpersonation(t *testing.T) {
	identity := func(kind ActorType) *_ActorPayload {
		return new(_ActorPayload{
			Type:         kind,
			Realm:        "demo.UserActor",
			Identifier:   "user",
			InfoSkelName: "demo.UserActorInfo",
			Info:         []byte(`{"id":"user"}`),
		})
	}
	nested := identity(ActorTypeImpersonated)
	nested.Impersonator = identity(ActorTypeAuthenticated)
	hiddenNested := identity(ActorTypeAuthenticated)
	hiddenNested.Impersonator = identity(ActorTypeAuthenticated)
	for _, tc := range []struct {
		name     string
		operator *_ActorPayload
	}{
		{name: "missing"},
		{name: "nested", operator: nested},
		{name: "nested with authenticated type", operator: hiddenNested},
		{name: "impersonated without operator", operator: identity(ActorTypeImpersonated)},
		{name: "anonymous", operator: identity(ActorTypeAnonymous)},
		{name: "authenticating", operator: identity(ActorTypeAuthenticating)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := identity(ActorTypeImpersonated)
			payload.Impersonator = tc.operator
			_, err := DecodeActorFromBase64(encodePayloadToBase64(payload))
			if err == nil || !strings.Contains(err.Error(), "impersonator must be ordinarily authenticated") {
				t.Fatalf("expected operator validation error, got %v", err)
			}
		})
	}
}
