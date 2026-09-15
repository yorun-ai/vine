package syncer

import (
	"encoding/json/jsontext"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

type Syncer struct {
	app.BaseModule

	WatchServer *watchserver.Server `inject:""`

	namesMutex           sync.Mutex
	schemaMutex          sync.Mutex
	appConfigNamesById   map[int]string
	portalSiteNamesById  map[int]string
	portalRuleNamesById  map[int]string
	portalRulesById      map[int]*core.PortalRule
	portalCertNamesById  map[int]string
	schemaActorHashes    map[string]string
	schemaResourceHashes map[string]string
	schemaServiceHashes  map[string]string
}

func (s *Syncer) DIInit() {
	s.appConfigNamesById = map[int]string{}
	s.portalSiteNamesById = map[int]string{}
	s.portalRuleNamesById = map[int]string{}
	s.portalRulesById = map[int]*core.PortalRule{}
	s.portalCertNamesById = map[int]string{}
	s.schemaActorHashes = map[string]string{}
	s.schemaResourceHashes = map[string]string{}
	s.schemaServiceHashes = map[string]string{}
}

func (s *Syncer) SyncAppConfig(item *core.AppConfig) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removeRenamedKeyLocked(s.appConfigNamesById, item.Id, item.Name, watched.FormatConfigKey)
	s.WatchServer.SetAndNotify(watched.FormatConfigKey(item.Name), vcode.MustMarshalJsonS(ToWatchedAppConfig(item)))
	s.saveNameByIdLocked(s.appConfigNamesById, item.Id, item.Name)
}

func (s *Syncer) RemoveAppConfig(item *core.AppConfig) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatConfigKey(item.Name))
	delete(s.appConfigNamesById, item.Id)
}

func (s *Syncer) SyncRpcServiceRegistration(reg watched.RpcServiceRegistration) {
	s.WatchServer.SetAndNotify(
		watched.FormatRpcServiceRegistrationKey(reg.ServiceName, reg.AppName, reg.AppInstanceId),
		vcode.MustMarshalJsonS(reg),
	)
}

func (s *Syncer) SyncWebRegistration(webName string, reg watched.WebRegistration) {
	s.WatchServer.SetAndNotify(
		watched.FormatWebRegistrationKey(webName, reg.AppName, reg.AppInstanceId),
		vcode.MustMarshalJsonS(reg),
	)
}

func (s *Syncer) removeRenamedKeyLocked(namesById map[int]string, id int, name string, formatKey func(string) string) {
	if oldName, ok := namesById[id]; ok && oldName != name {
		s.WatchServer.DeleteAndNotify(formatKey(oldName))
	}
}

func (s *Syncer) saveNameByIdLocked(namesById map[int]string, id int, name string) {
	namesById[id] = name
}

func ToWatchedAppConfig(item *core.AppConfig) *watched.ConfigValue {
	return &watched.ConfigValue{
		Name:  item.Name,
		Value: jsontext.Value(item.Value),
	}
}
