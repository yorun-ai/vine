package app

import (
	"reflect"

	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/meta"
)

// T returns the reflect.Type for T and keeps app package type references concise.
func T[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// bindActor bind meta.Actor and registered ActorInfo type
func bindActor(b *di.Binder) {
	b.BindFactory(func(ctx meta.Context) meta.Actor {
		return ctx.Actor()
	})
	for _, actorInfoType := range meta.RegisteredActorInfoTypes() {
		b.Bind(actorInfoType).ToFactory(func(ctx meta.Context) any {
			info, _ := meta.GetActorInfoByType(ctx.Actor(), actorInfoType)
			return info
		}).AsNullable()
	}
}
