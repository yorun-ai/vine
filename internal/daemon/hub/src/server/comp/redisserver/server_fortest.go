package redisserver

import (
	"context"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver/embedded"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func NewServerForTest() *Server {
	server := &Server{
		Context:    context.Background(),
		Option:     &flag.Flag{},
		InprocFlag: &app.InternalInprocFlag{},
	}
	server.serverPassword = newServerPassword()
	server.store = embedded.NewStore(server.Option.RedisListen, server.InprocFlag.Enabled, server.serverPassword)
	server.store.InitRevision()
	return server
}

func SetTimeNowForTest(t *testing.T, now func() time.Time) {
	t.Helper()
	old := timeNow
	timeNow = now
	t.Cleanup(func() {
		timeNow = old
	})
	embedded.SetTimeNowForTest(t, now)
}

func RedisListenAddrForTest(t *testing.T, server *Server) string {
	t.Helper()
	store, ok := server.store.(*embedded.Store)
	if !ok {
		t.Fatal("redis test server must use embedded store")
	}
	return store.ListenAddr()
}
