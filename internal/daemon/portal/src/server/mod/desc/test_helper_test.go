package desc

import (
	"testing"

	"go.yorun.ai/vine/internal/utilfortest/watchtest"
)

func newTestManager(t *testing.T, values map[string]string) *Manager {
	t.Helper()
	manager := new(Manager{Context: t.Context(), Watch: watchtest.New(t, values)})
	manager.DIInit()
	t.Cleanup(manager.AfterAppStop)
	return manager
}
