package meta_test

import (
	"testing"

	"go.yorun.ai/vine/core/meta"
	internalmeta "go.yorun.ai/vine/internal/core/meta"
)

func TestImpersonatedActorFacade(t *testing.T) {
	effective := internalmeta.NewAuthenticatedActorWithRawInfo("demo.User", "user", "demo.UserInfo", []byte(`{"id":"user"}`))
	operator := internalmeta.NewAuthenticatedActorWithRawInfo("demo.Ops", "ops", "demo.OpsInfo", []byte(`{"id":"ops"}`))
	actor := meta.NewImpersonatedActor(effective, operator)
	if !actor.IsAuthenticated() || !actor.IsImpersonated() || actor.Type() != meta.ActorTypeImpersonated {
		t.Fatal("unexpected actor state")
	}
	if actor.Realm() != effective.Realm() || actor.Identifier() != effective.Identifier() || actor.RawInfo() != effective.RawInfo() || actor.Impersonator() != operator {
		t.Fatal("identity was not preserved")
	}
	fields := map[string]string{}
	for _, attr := range actor.LogFields() {
		fields[attr.Key] = attr.Value.String()
	}
	if len(fields) != 5 || fields["actorType"] != "impersonated" || fields["actorRealm"] != "demo.User" || fields["actorIdentifier"] != "user" || fields["impersonatorActorRealm"] != "demo.Ops" || fields["impersonatorActorIdentifier"] != "ops" {
		t.Fatalf("unexpected actor log fields: %#v", fields)
	}
}
