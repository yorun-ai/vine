package meta

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"go.yorun.ai/vine/util/vstring"
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

var applicationNamePattern = regexp.MustCompile(`^[a-z]+(?:\.[a-z]+)*(?:@[a-z]+(?:\.[a-z]+)*)?$`)

func NewApp(name string, version string, instanceId string) (App, error) {
	if !IsValidName(name) {
		return nil, fmt.Errorf("invalid name, lowercase letters and dots expected, optional @suffix supported")
	}
	if !IsValidVersion(version) {
		return nil, fmt.Errorf("invalid version, semantic expected")
	}
	if !IsValidInstanceId(instanceId) {
		return nil, fmt.Errorf("invalid instanceId, uuid expected")
	}
	return &_App{
		name:       name,
		version:    version,
		instanceId: instanceId,
	}, nil
}

func DecodeAppFromDelimited(value string) (App, error) {
	fields, err := vstring.DecodeDelimited(value)
	if err != nil {
		return nil, err
	}

	name := fields["name"]
	version := fields["version"]
	instanceId := fields["instanceId"]
	if name == "" || version == "" || instanceId == "" {
		return nil, fmt.Errorf("missing app field")
	}

	return NewApp(name, version, instanceId)
}

func EncodeAppToDelimited(app App) string {
	return vstring.EncodeDelimited(
		"name", app.Name(),
		"version", app.Version(),
		"instanceId", app.InstanceId(),
	)
}

func MustNewApp(name string, version string, instanceId string) App {
	app, err := NewApp(name, version, instanceId)
	if err != nil {
		panic(err)
	}
	return app
}

// MustNewInstanceId creates a process/app instance UUID using V7 for time-ordered IDs.
func MustNewInstanceId() string {
	return uuid.Must(uuid.NewV7()).String()
}

func MustNewAppWithRandomId(name string, version string) App {
	return MustNewApp(name, version, MustNewInstanceId())
}

func (a _App) Name() string {
	return a.name
}

func (a _App) Version() string {
	return a.version
}

func (a _App) InstanceId() string {
	return a.instanceId
}

func IsSame(left App, right App) bool {
	if left == nil || right == nil {
		return false
	}
	return left.Name() == right.Name() && left.Version() == right.Version() && left.InstanceId() == right.InstanceId()
}

func IsValidName(name string) bool {
	return applicationNamePattern.MatchString(name)
}

func IsValidVersion(version string) bool {
	_, err := semver.NewVersion(version)
	return err == nil
}

func IsValidInstanceId(instanceId string) bool {
	_, err := uuid.Parse(instanceId)
	return err == nil
}
