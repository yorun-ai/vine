package syncer

import (
	"fmt"
	"sync"
	"testing"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestSyncerSupportsConcurrentStateUpdates(t *testing.T) {
	redisServer := redisserver.NewServerForTest()
	defer redisServer.AfterAppStop()
	target := testSyncer(redisServer)

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		waitGroup.Go(func() {
			<-start
			for iteration := 0; iteration < 20; iteration++ {
				id := iteration % 3
				name := fmt.Sprintf("demo.%d.%d", worker, iteration)
				target.SyncAppConfig(&core.AppConfig{Id: id, Name: name, Value: `{}`})
				target.SyncPortalSiteWithRpcgwServices(&core.PortalSite{Id: id, Name: name, Type: core.PortalSiteTypeWEBGW}, nil)
				target.SyncPortalRule(&core.PortalRule{Id: id, Name: name})
				target.SyncPortalCert(&core.PortalCert{Id: id, Name: name})

				views := concurrentSchemaViews(name)
				if iteration%2 == 0 {
					target.SyncSchemas(views)
				} else {
					target.WriteSchemas(views)
				}
			}
		})
	}
	close(start)
	waitGroup.Wait()
}

func concurrentSchemaViews(name string) []core.DomainSchemaView {
	return []core.DomainSchemaView{{
		Actors: []core.SchemaVersion[*skel.ActorSchema]{{
			Schema:     &skel.ActorSchema{SkelName: name, Hash: name},
			SkelName:   name,
			SchemaHash: name,
			Main:       true,
		}},
	}}
}
