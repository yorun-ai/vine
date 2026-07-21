package meta

import "reflect"

type _AuthenticatedActorInfoForTest struct {
	Id string
}

func NewAuthenticatedActorForTest() Actor {
	infoType := reflect.TypeFor[*_AuthenticatedActorInfoForTest]()
	actorInfo := infoByInfoType[infoType]
	if actorInfo == nil {
		RegisterActor(ActorSpec{
			Name:         "AuthenticatedActorForTest",
			SkelName:     "test.meta.AuthenticatedActorForTest",
			InfoSkelName: "test.meta.AuthenticatedActorInfoForTest",
			InfoType:     infoType,
		})
		actorInfo = infoByInfoType[infoType]
	}
	return &_Actor{
		kind:        ActorTypeAuthenticated,
		actorInfo:   actorInfo,
		rawAuthInfo: []byte(`{"Id":"test"}`),
	}
}
