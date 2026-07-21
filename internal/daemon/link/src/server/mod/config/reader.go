package config

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/link/src/server/mod/minder"
)

type _InstantConfigState struct {
	refsByAppInstanceID map[string]struct{}
	value               string
	cancel              context.CancelFunc
}

type Reader struct {
	app.BaseModule

	Context   context.Context   `inject:""`
	Client    *hubredis.Client  `inject:""`
	AppMinder *minder.AppMinder `inject:""`

	mutex                       sync.RWMutex
	instantConfigStatesByKey    map[string]*_InstantConfigState
	configValuesByAppInstanceID map[string]map[string]string
}

func (c *Reader) DIInit() {
	c.instantConfigStatesByKey = map[string]*_InstantConfigState{}
	c.configValuesByAppInstanceID = map[string]map[string]string{}
	c.AppMinder.AddMutator(c)
}

func (c *Reader) AfterAppStop() {
	c.stopAllInstantConfigWatchers()
}

func (c *Reader) OnSetup(instance *minder.AppInstance) {
	c.mutex.Lock()
	if _, ok := c.configValuesByAppInstanceID[instance.AppInfo.InstanceId()]; !ok {
		c.configValuesByAppInstanceID[instance.AppInfo.InstanceId()] = map[string]string{}
	}
	c.mutex.Unlock()
}

func (*Reader) OnDrain(*minder.AppInstance) {}

func (c *Reader) OnDestroy(instance *minder.AppInstance) {
	c.releaseInstantConfigStateByInstance(instance.AppInfo.InstanceId())
}
