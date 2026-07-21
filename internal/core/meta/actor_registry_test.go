package meta

import (
	"reflect"
	"testing"
)

type _RegistryTestActorInfo struct {
	Subject string `json:"subject"`
}

func TestRegisterActorStoresActorSpec(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:         "RegistryTestActor",
		SkelName:     "test.registry.RegistryTestActor",
		Hash:         "hash-registry-test",
		InfoSkelName: "test.registry.RegistryTestActorInfo",
		InfoType:     reflect.TypeFor[*_RegistryTestActorInfo](),
	}

	RegisterActor(spec)

	got, ok := infoBySkelName[spec.SkelName]
	if !ok {
		t.Fatalf("expected actor info")
	}
	assertActorInfo(t, got, spec)

	infoByData, ok := infoByInfoSkelName[spec.InfoSkelName]
	if !ok {
		t.Fatalf("expected actor info by data skel name")
	}
	assertActorInfo(t, infoByData, spec)
	infoByType, ok := infoByInfoType[spec.InfoType]
	if !ok {
		t.Fatalf("expected actor info by info type")
	}
	assertActorInfo(t, infoByType, spec)
}

func TestRegisterActorAllowsActorWithoutInfo(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:     "RegistryNoInfoActor",
		SkelName: "test.registry.RegistryNoInfoActor",
		Hash:     "hash-registry-no-info",
	}

	RegisterActor(spec)

	got, ok := infoBySkelName[spec.SkelName]
	if !ok {
		t.Fatalf("expected actor info")
	}
	if got.InfoSkelName != "" || got.InfoType != nil {
		t.Fatalf("unexpected actor info data: %#v", got)
	}
}

func TestRegisterActorPanicsOnDuplicateActorSkelName(t *testing.T) {
	resetActorRegistryForTest()

	spec := ActorSpec{
		Name:     "RegistryDuplicateActor",
		SkelName: "test.registry.RegistryDuplicateActor",
	}
	RegisterActor(spec)

	assertPanics(t, func() {
		RegisterActor(spec)
	})
}

func TestRegisterActorPanicsOnInvalidInfoType(t *testing.T) {
	resetActorRegistryForTest()

	assertPanics(t, func() {
		RegisterActor(ActorSpec{
			Name:         "RegistryInvalidInfoActor",
			SkelName:     "test.registry.RegistryInvalidInfoActor",
			InfoSkelName: "test.registry.RegistryInvalidInfoActorInfo",
			InfoType:     reflect.TypeFor[_RegistryTestActorInfo](),
		})
	})
}

func resetActorRegistryForTest() {
	infoBySkelName = map[string]*_ActorInfo{}
	infoByInfoSkelName = map[string]*_ActorInfo{}
	infoByInfoType = map[reflect.Type]*_ActorInfo{}
}

func assertActorInfo(t *testing.T, info *_ActorInfo, spec ActorSpec) {
	t.Helper()

	if info.Name != spec.Name {
		t.Fatalf("unexpected actor name: got=%s want=%s", info.Name, spec.Name)
	}
	if info.SkelName != spec.SkelName {
		t.Fatalf("unexpected actor skelName: got=%s want=%s", info.SkelName, spec.SkelName)
	}
	if info.Hash != spec.Hash {
		t.Fatalf("unexpected actor hash: got=%s want=%s", info.Hash, spec.Hash)
	}
	if info.InfoSkelName != spec.InfoSkelName {
		t.Fatalf("unexpected actor info skelName: got=%s want=%s", info.InfoSkelName, spec.InfoSkelName)
	}
	if info.InfoType != spec.InfoType {
		t.Fatalf("unexpected actor info type: got=%v want=%v", info.InfoType, spec.InfoType)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}
