package log

import "testing"

func resetSettingsForTest(t *testing.T) {
	t.Helper()

	prevInprocClientLogEnabled := IsInprocClientLogEnabled()
	DisableInprocClientLog()
	t.Cleanup(func() {
		if prevInprocClientLogEnabled {
			EnableInprocClientLog()
		} else {
			DisableInprocClientLog()
		}
	})
}

func TestSettingsDefaultDisabled(t *testing.T) {
	resetSettingsForTest(t)

	if IsInprocClientLogEnabled() {
		t.Fatal("expected inproc client log to be disabled")
	}
}

func TestEnableDisableInprocClientLog(t *testing.T) {
	resetSettingsForTest(t)

	EnableInprocClientLog()
	if !IsInprocClientLogEnabled() {
		t.Fatal("expected inproc client log to be enabled")
	}

	DisableInprocClientLog()
	if IsInprocClientLogEnabled() {
		t.Fatal("expected inproc client log to be disabled")
	}
}
