package runtime

import (
	"go.yorun.ai/vine/buildinfo"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vpre"
)

type App interface {
	Name() string
	Version() string
	InstanceId() string
}

type _App struct {
	name       string
	version    string
	instanceId string
}

var (
	runtimeName    = buildinfo.Name
	runtimeVersion = buildinfo.Version
	// currentApp is the process-level singleton app identity for the running vine app.
	// It is initialized during package init, can be configured during startup,
	// and becomes immutable after Application() is first called.
	currentApp  = _App{}
	appAccessed = false
)

func init() {
	name, _ := runtimeName()
	vpre.Check(meta.IsValidName(name), "invalid runtime.ldName: %q, lowercase letters and dots expected", name)
	currentApp.name = name

	version, _ := runtimeVersion()
	vpre.Check(meta.IsValidVersion(version), "invalid runtime.ldVersion: %q, semantic version expected", version)
	currentApp.version = version

	currentApp.instanceId = meta.MustNewInstanceId()
}

func Application() App {
	appAccessed = true
	return &currentApp
}

func SetName(name string) {
	vpre.Check(!appAccessed, "application already accessed")
	if _, ok := runtimeName(); !ok {
		vpre.Check(meta.IsValidName(name), "invalid name: %q, lowercase letters and dots expected", name)
		currentApp.name = name
	}
}

func SetVersion(version string) {
	vpre.Check(!appAccessed, "application already accessed")
	if _, ok := runtimeVersion(); !ok {
		vpre.Check(meta.IsValidVersion(version), "invalid version: %q, semantic version expected", version)
		currentApp.version = version
	}
}

func SetInstanceId(instanceId string) {
	vpre.Check(!appAccessed, "application already accessed")
	vpre.Check(meta.IsValidInstanceId(instanceId), "invalid instanceId: %q, uuid expected", instanceId)
	currentApp.instanceId = instanceId
}

func (a *_App) Name() string {
	return a.name
}

func (a *_App) Version() string {
	return a.version
}

func (a *_App) InstanceId() string {
	return a.instanceId
}
