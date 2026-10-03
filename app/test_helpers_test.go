package app

import (
	"go.yorun.ai/vine/internal/utilfortest/linktest"
	"os"
	"testing"

	"go.yorun.ai/vine/internal/core/link"
	"go.yorun.ai/vine/internal/core/meta"
)

func TestMain(m *testing.M) {
	restoreLinkerFactory := link.SetNewLinkerForTest(func(_ meta.App, _ string) link.Linker {
		return &linktest.Linker{
			RpcProxyOutEndpointValue: "http://127.0.0.1:7079/rpc/proxy/out",
		}
	})
	code := m.Run()
	restoreLinkerFactory()
	os.Exit(code)
}
