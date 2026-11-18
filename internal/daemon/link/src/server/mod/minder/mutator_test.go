package minder

import (
	"context"
	"sync"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type _Mutator struct {
	mutex        sync.Mutex
	setupIDs     []uuid.UUID
	drainedIDs   []uuid.UUID
	destroyedIDs []uuid.UUID
}

func (w *_Mutator) OnSetup(instance *AppInstance) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.setupIDs = append(w.setupIDs, instance.AppInfo.InstanceId())
}

func (w *_Mutator) OnDrain(instance *AppInstance) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.drainedIDs = append(w.drainedIDs, instance.AppInfo.InstanceId())
}

func (w *_Mutator) OnDestroy(instance *AppInstance) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.destroyedIDs = append(w.destroyedIDs, instance.AppInfo.InstanceId())
}

func TestDrainAndDestroyNotifyMutatorsOnce(t *testing.T) {
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		CurrentApp:            mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: &_RegistryServiceClient{},
		HubInfo:               newTestHubInfo(new(_TestInfoServiceClient)),
	}
	minder.DIInit()
	mutator := &_Mutator{}
	minder.AddMutator(mutator)
	instance := minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})
	assert.True(t, minder.addInstance(instance))
	minder.beforeRegistration(instance)

	minder.removeInstance(instance)
	minder.onDrain(instance)
	minder.onDestroy(instance)
	minder.removeInstance(instance)

	mutator.mutex.Lock()
	defer mutator.mutex.Unlock()
	assert.Equal(t, []uuid.UUID{mustTestMetaApp().InstanceId()}, mutator.setupIDs)
	assert.Equal(t, []uuid.UUID{mustTestMetaApp().InstanceId()}, mutator.drainedIDs)
	assert.Equal(t, []uuid.UUID{mustTestMetaApp().InstanceId()}, mutator.destroyedIDs)
}

func TestAddInstanceRejectsDuplicateInstanceID(t *testing.T) {
	minder := &AppMinder{
		Context:               context.Background(),
		Flag:                  &flag.Flag{},
		CurrentApp:            mustTestMetaApp(),
		InprocFlag:            &app.InternalInprocFlag{},
		RegistryServiceClient: &_RegistryServiceClient{},
		HubInfo:               newTestHubInfo(new(_TestInfoServiceClient)),
	}
	minder.DIInit()

	assert.True(t, minder.addInstance(minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})))
	assert.False(t, minder.addInstance(minder.newAppInstance(AppRegistration{AppInfo: mustTestMetaApp()})))
}
