package meta

import (
	"fmt"
	"regexp"
	"strings"
	"uuid"

	"github.com/Masterminds/semver/v3"
	"go.yorun.ai/vine/util/vstring"
)

type App interface {
	Name() string
	Version() string
	InstanceId() uuid.UUID
}

type _App struct {
	name       string
	version    string
	instanceId uuid.UUID
}

// applicationNamePattern matches an application name: lowercase letters and
// digits in dot-separated segments, each starting with a letter. It is the same
// shape a Plot domain name has, because a domain name becomes the name of the
// application that serves it.
var applicationNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)*$`)

func NewApp(name string, version string, instanceId uuid.UUID) (App, error) {
	if !IsValidName(name) {
		return nil, fmt.Errorf("invalid name, lowercase letters, digits and dots expected")
	}
	if !IsValidVersion(version) {
		return nil, fmt.Errorf("invalid version, semantic expected")
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

	id, err := uuid.Parse(instanceId)
	if err != nil {
		return nil, fmt.Errorf("invalid instanceId: %w", err)
	}
	return NewApp(name, version, id)
}

func EncodeAppToDelimited(app App) string {
	return vstring.EncodeDelimited(
		"name", app.Name(),
		"version", app.Version(),
		"instanceId", app.InstanceId().String(),
	)
}

func MustNewApp(name string, version string, instanceId uuid.UUID) App {
	app, err := NewApp(name, version, instanceId)
	if err != nil {
		panic(err)
	}
	return app
}

// MustNewInstanceId creates a process/app instance UUID using V7 for time-ordered IDs.
func MustNewInstanceId() uuid.UUID {
	return uuid.NewV7()
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

func (a _App) InstanceId() uuid.UUID {
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

// IsValidVersion reports whether version is usable as application metadata. A
// version carries the Go module form, so a leading "v" is accepted; everything
// after it must be a full semantic version, which is what the Rpc identity
// header requires.
func IsValidVersion(version string) bool {
	_, err := semver.StrictNewVersion(strings.TrimPrefix(version, "v"))
	return err == nil
}

func IsValidInstanceId(instanceId string) bool {
	_, err := uuid.Parse(instanceId)
	return err == nil
}
