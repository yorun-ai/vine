package hubinfo

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	hubskeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

type _TestInfoServiceClient struct {
	info        hubskeled.Info
	getInfoCall int
}

func (c *_TestInfoServiceClient) GetInfo(_ ...rpcclient.InvokeOption) hubskeled.Info {
	c.getInfoCall++
	return c.info
}

func TestHubInfoDIInitLoadsHubInfo(t *testing.T) {
	client := &_TestInfoServiceClient{info: hubskeled.Info{WatchPort: 7072}}
	flags := &flag.Flag{HubEndpoint: "http://[::1]:7071"}
	flags.Normalize()
	component := &HubInfo{Flag: flags, InfoServiceClient: client}

	component.DIInit()
	assert.Equal(t, 1, client.getInfoCall)
	assert.Equal(t, "[::1]:7072", component.WatchEndpoint())
}

func TestHubInfoDIInitSkipsHubInfoLookupInInprocMode(t *testing.T) {
	client := &_TestInfoServiceClient{}
	flags := &flag.Flag{HubInprocMode: true}
	flags.Normalize()
	component := &HubInfo{Flag: flags, InfoServiceClient: client}

	component.DIInit()
	assert.Equal(t, 0, client.getInfoCall)
}

func TestHubInfoRefreshNotifiesListenersOnlyWhenHubInfoChanged(t *testing.T) {
	client := &_TestInfoServiceClient{info: hubskeled.Info{WatchPort: 7072}}
	flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
	flags.Normalize()
	component := &HubInfo{Flag: flags, InfoServiceClient: client}
	component.DIInit()

	refreshes := 0
	component.OnRefresh(func() { refreshes++ })

	component.Refresh()
	assert.Equal(t, 2, client.getInfoCall)
	assert.Equal(t, 0, refreshes)

	client.info = hubskeled.Info{WatchPort: 7073}
	component.Refresh()

	assert.Equal(t, 1, refreshes)
	assert.Equal(t, "127.0.0.1:7073", component.WatchEndpoint())
}

func TestHubInfoBeforeAppStartPollsHubInformation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &_TestInfoServiceClient{info: hubskeled.Info{WatchPort: 7072}}
		flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
		flags.Normalize()
		component := &HubInfo{Context: t.Context(), Flag: flags, InfoServiceClient: client}
		component.DIInit()

		prev := refreshInterval
		refreshInterval = 10 * time.Millisecond
		defer func() { refreshInterval = prev }()

		require.NoError(t, component.BeforeAppStart())
		defer component.AfterAppStop()

		refreshes := 0
		component.OnRefresh(func() { refreshes++ })
		synctest.Sleep(refreshInterval)

		assert.GreaterOrEqual(t, client.getInfoCall, 2)
		assert.Equal(t, 0, refreshes)
	})
}

func TestConcurrentRefreshAfterFailedRepairKeepsLatestEndpoint(t *testing.T) {
	for _, port := range []int{7072, 7074} {
		client := &_TestInfoServiceClient{info: hubskeled.Info{WatchPort: 7072}}
		flags := &flag.Flag{HubEndpoint: "http://127.0.0.1:7071"}
		flags.Normalize()
		component := &HubInfo{Flag: flags, InfoServiceClient: client}
		component.DIInit()

		entered := make(chan struct{})
		release := make(chan struct{})
		var first sync.Once
		component.OnRefresh(func() {
			first.Do(func() {
				close(entered)
				<-release
				panic("repair failed")
			})
		})
		repairs := make(chan string, 2)
		component.OnRefresh(func() { repairs <- component.WatchEndpoint() })
		client.info.WatchPort = 7073
		failure := make(chan any, 1)
		go func() {
			defer func() { failure <- recover() }()
			component.Refresh()
		}()
		<-entered
		client.info.WatchPort = port
		starting := make(chan struct{})
		done := make(chan struct{})
		go func() {
			close(starting)
			component.Refresh()
			close(done)
		}()
		<-starting
		close(release)
		assert.Equal(t, "repair failed", <-failure)
		<-done
		close(repairs)
		var repaired []string
		for endpoint := range repairs {
			repaired = append(repaired, endpoint)
		}
		want := fmt.Sprintf("127.0.0.1:%d", port)
		assert.Equal(t, want, component.WatchEndpoint())
		assert.Equal(t, []string{"127.0.0.1:7073", want}, repaired)
	}
}
