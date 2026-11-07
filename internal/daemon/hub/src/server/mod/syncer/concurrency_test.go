package syncer

import (
	"fmt"
	"sync"
	"testing"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestSyncerSupportsConcurrentStateUpdates(t *testing.T) {
	watchServer := watchserver.NewServerForTest()
	defer watchServer.AfterAppStop()
	target := testSyncer(watchServer)

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for worker := range 4 {
		waitGroup.Go(func() {
			<-start
			for iteration := range 20 {
				id := iteration % 3
				name := fmt.Sprintf("demo.%d.%d", worker, iteration)
				target.SyncAppConfig(&core.AppConfig{Id: id, Name: name, Value: `{}`})
				target.SyncPortalSite(&core.PortalSite{Id: id, Name: name, Type: core.PortalSiteTypeWEBGW})
				target.SyncPortalRule(&core.PortalRule{Id: id, Name: name})
				target.SyncPortalCert(&core.PortalCert{Id: id, Name: name})

				views := concurrentDescriptorViews(name)
				if iteration%2 == 0 {
					target.SyncDescriptors(views)
				} else {
					target.WriteDescriptors(views)
				}
			}
		})
	}
	close(start)
	waitGroup.Wait()
}

func concurrentDescriptorViews(name string) []core.DomainDescriptorView {
	return []core.DomainDescriptorView{{
		Actors: []core.DescriptorVersion[*skeldesc.Actor]{{
			Descriptor:     &skeldesc.Actor{SkelName: name, Hash: name},
			SkelName:       name,
			DescriptorHash: name,
			Main:           true,
		}},
	}}
}
