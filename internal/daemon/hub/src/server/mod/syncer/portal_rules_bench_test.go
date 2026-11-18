package syncer

import (
	"fmt"
	"testing"

	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func BenchmarkPortalRuleSync(b *testing.B) {
	const count = 100
	rules := make([]*core.PortalRule, count)
	for i := range rules {
		rules[i] = &core.PortalRule{
			Id:                      i + 1,
			Name:                    fmt.Sprintf("rule-%03d", i),
			EntryId:                 1,
			Enabled:                 true,
			MatchPathPrefix:         fmt.Sprintf("/route/%d", i),
			RouteType:               core.PortalRuleRouteTypeTemporaryRedirect,
			RouteRedirectionPattern: "/v1",
		}
	}
	entry := &core.PortalEntry{Id: 1, Name: "web", Protocol: "http", Enabled: true}
	create := func() *Syncer {
		target := testSyncer(watchserver.NewServerForTest())
		target.SyncPortalEntry(entry)
		return target
	}
	b.Run("Load100", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			target := create()
			target.SyncPortalRules(rules)
			target.WatchServer.AfterAppStop()
		}
	})
	b.Run("UpdateRedirectOf100", func(b *testing.B) {
		target := create()
		defer target.WatchServer.AfterAppStop()
		target.SyncPortalRules(rules)
		rule := *rules[0]
		destinations := [2]string{"/v2", "/v1"}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			rule.RouteRedirectionPattern = destinations[i%2]
			target.SyncPortalRule(&rule)
			i++
		}
	})
}
