package lock

import (
	"context"
	"testing"

	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/link/src/server/comp/hubinfo"
	"go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

type infoClient struct{ info hubskeled.Info }

func (c *infoClient) GetInfo(...client.InvokeOption) hubskeled.Info {
	return c.info
}

type hubClient struct{ calls []string }

func (c *hubClient) Acquire(key string, token string, ttl int, opts ...client.InvokeOption) bool {
	c.calls = append(c.calls, "acquire")
	return true
}

func (c *hubClient) Renew(key string, token string, ttl int, opts ...client.InvokeOption) bool {
	c.calls = append(c.calls, "renew")
	return true
}

func (c *hubClient) Release(key string, token string, opts ...client.InvokeOption) bool {
	c.calls = append(c.calls, "release")
	return true
}

func newTestLocker(t *testing.T, mode string, endpoint string, inproc bool) (*Locker, *hubClient) {
	t.Helper()
	flags := &flag.Flag{HubEndpoint: "http://localhost:7071", HubInprocMode: inproc}
	flags.Normalize(false)
	info := &hubinfo.HubInfo{
		Flag:              flags,
		InfoServiceClient: &infoClient{info: hubskeled.Info{LockMode: mode, LockRedisEndpoint: endpoint}},
	}
	info.DIInit()
	hub := &hubClient{}
	locker := &Locker{HubInfo: info, Hub: hub}
	locker.DIInit()
	t.Cleanup(locker.AfterAppStop)
	return locker, hub
}

func newTestLockerWithHubInfo(t *testing.T, mode string, endpoint string) (*Locker, *infoClient, *hubinfo.HubInfo) {
	t.Helper()
	flags := &flag.Flag{HubEndpoint: "http://localhost:7071"}
	flags.Normalize(false)
	infoClientValue := &infoClient{info: hubskeled.Info{LockMode: mode, LockRedisEndpoint: endpoint}}
	info := &hubinfo.HubInfo{Flag: flags, InfoServiceClient: infoClientValue}
	info.DIInit()
	locker := &Locker{HubInfo: info, Hub: &hubClient{}}
	locker.DIInit()
	t.Cleanup(locker.AfterAppStop)
	return locker, infoClientValue, info
}

func lockContext(ctx context.Context) meta.Context {
	return meta.NewContext(ctx, meta.InitialTrace(), nil, meta.NewAbsentActor())
}
