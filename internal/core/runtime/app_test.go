package runtime

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/core/meta"
)

func withRuntimeOverride(name func() (string, bool), version func() (string, bool), fn func()) {
	prevName := runtimeName
	prevVersion := runtimeVersion
	runtimeName = name
	runtimeVersion = version
	defer func() {
		runtimeName = prevName
		runtimeVersion = prevVersion
	}()
	fn()
}

func resetApplicationState() {
	appAccessed = false
	currentApp = _App{}
}

func TestApplicationUsesSettersAndGeneratedInstanceID(t *testing.T) {
	resetApplicationState()
	SetName("user.service")
	SetVersion("1.2.3")
	SetInstanceId(meta.MustNewInstanceId())
	app := Application()

	assert.Equal(t, "user.service", app.Name())
	assert.Equal(t, "1.2.3", app.Version())
	assert.NotEmpty(t, app.InstanceId())
	assert.NoError(t, uuid.Validate(app.InstanceId()))
	parsed := uuid.MustParse(app.InstanceId())
	if parsed.Version() != 7 {
		t.Fatalf("expected v7 uuid, got v%d", parsed.Version())
	}
}

func TestSetNamePanicsOnInvalidName(t *testing.T) {
	resetApplicationState()
	assert.PanicsWithError(t,
		`invalid name: "UserService", lowercase letters and dots expected`,
		func() {
			SetName("UserService")
		},
	)
}

func TestSetNamePanicsAfterApplicationAccessed(t *testing.T) {
	resetApplicationState()
	_ = Application()
	assert.PanicsWithError(t,
		`application already accessed`,
		func() {
			SetName("other.service")
		},
	)
}

func TestSetVersionPanicsOnInvalidVersion(t *testing.T) {
	resetApplicationState()
	assert.PanicsWithError(t,
		`invalid version: "invalid", semantic version expected`,
		func() {
			SetVersion("invalid")
		},
	)
}

func TestSetVersionPanicsAfterApplicationAccessed(t *testing.T) {
	resetApplicationState()
	_ = Application()
	assert.PanicsWithError(t,
		`application already accessed`,
		func() {
			SetVersion("2.0.0")
		},
	)
}

func TestSetInstanceIdPanicsOnInvalidInstanceID(t *testing.T) {
	resetApplicationState()
	assert.PanicsWithError(t,
		`invalid instanceId: "bad-id", uuid expected`,
		func() {
			SetInstanceId("bad-id")
		},
	)
}

func TestSetInstanceIdPanicsAfterApplicationAccessed(t *testing.T) {
	resetApplicationState()
	_ = Application()
	assert.PanicsWithError(t,
		`application already accessed`,
		func() {
			SetInstanceId(meta.MustNewInstanceId())
		},
	)
}

func TestSetNameIgnoredWhenRuntimeNameOverrides(t *testing.T) {
	resetApplicationState()
	withRuntimeOverride(
		func() (string, bool) { return "ld.name", true },
		runtimeVersion,
		func() {
			SetName("user.service")
			assert.Empty(t, currentApp.name)
		},
	)
}

func TestSetVersionIgnoredWhenRuntimeVersionOverrides(t *testing.T) {
	resetApplicationState()
	withRuntimeOverride(
		runtimeName,
		func() (string, bool) { return "9.9.9", true },
		func() {
			SetVersion("1.2.3")
			assert.Empty(t, currentApp.version)
		},
	)
}

func TestValidationHelpers(t *testing.T) {
	assert.True(t, meta.IsValidName("user.service"))
	assert.True(t, meta.IsValidName("user.worker@runtime.app"))
	assert.False(t, meta.IsValidName(""))
	assert.False(t, meta.IsValidName("user-service"))
	assert.False(t, meta.IsValidName("User.Service"))

	assert.True(t, meta.IsValidVersion("1.2.3"))
	assert.False(t, meta.IsValidVersion("invalid"))

	validID := meta.MustNewInstanceId()
	assert.True(t, meta.IsValidInstanceId(validID))
	assert.False(t, meta.IsValidInstanceId("bad-id"))
}
