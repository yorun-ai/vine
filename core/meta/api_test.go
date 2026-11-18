package meta_test

import (
	"testing"
	"uuid"

	"go.yorun.ai/vine/core/meta"
	internalmeta "go.yorun.ai/vine/internal/core/meta"
)

func TestAppUUIDFacade(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	app, err := meta.NewApp("demo.service", "1.2.3", id)
	if err != nil {
		t.Fatal(err)
	}
	var current meta.CurrentApp = app
	instances := map[uuid.UUID]meta.App{current.InstanceId(): app}
	if instances[id] != app {
		t.Fatal("instance UUID was not preserved")
	}
	initiator, err := meta.NewInitiator(app.Name(), app.Version(), id, "https", "127.0.0.1")
	if err != nil || initiator.InstanceId() != id {
		t.Fatalf("initiator UUID: %v, %v", initiator, err)
	}
}

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
