package app

import (
	"context"
	"reflect"
	"testing"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/meta"
)

type _UtilTestActorInfoA struct {
	Name string
}

type _UtilTestActorInfoB struct {
	Name string
}

type _UtilTestActorInfoConsumer struct {
	A *_UtilTestActorInfoA `inject:""`
	B *_UtilTestActorInfoB `inject:""`
}

func TestBindActorAllowsNullableUnmatchedActorInfos(t *testing.T) {
	meta.RegisterActor(meta.ActorSpec{
		Name:         "UtilTestActorA",
		SkelName:     "test.app.UtilTestActorA",
		InfoSkelName: "test.app.UtilTestActorAInfo",
		InfoType:     reflect.TypeFor[*_UtilTestActorInfoA](),
	})
	meta.RegisterActor(meta.ActorSpec{
		Name:         "UtilTestActorB",
		SkelName:     "test.app.UtilTestActorB",
		InfoSkelName: "test.app.UtilTestActorBInfo",
		InfoType:     reflect.TypeFor[*_UtilTestActorInfoB](),
	})

	actor, err := meta.DecodeActorFromBase64(meta.EncodeActorToBase64(meta.NewAuthenticatedActorWithRawInfo(
		"test.app.UtilTestActorAInfo",
		[]byte(`{"Name":"A"}`),
	)))
	if err != nil {
		t.Fatalf("DecodeActorFromBase64() error = %v", err)
	}
	ctx := meta.NewContext(context.Background(), nil, nil, actor)

	injector := di.NewInjector(func(b *di.Binder) {
		b.Bind(di.T[meta.Context]()).ToInstance(ctx)
		bindActor(b)
		b.Bind(di.T[*_UtilTestActorInfoConsumer]())
	})

	var consumer *_UtilTestActorInfoConsumer
	injector.Resolve(&consumer)

	if consumer.A == nil || consumer.A.Name != "A" {
		t.Fatalf("unexpected actor info A: %#v", consumer.A)
	}
	if consumer.B != nil {
		t.Fatalf("unexpected actor info B: %#v", consumer.B)
	}
}
