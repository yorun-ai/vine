package syncer

import (
	"encoding/json"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

type Syncer struct {
	app.BaseModule

	RedisServer *redisserver.Server `inject:""`

	appConfigNamesById   map[int]string
	portalSiteNamesById  map[int]string
	portalRuleNamesById  map[int]string
	portalCertNamesById  map[int]string
	schemaActorHashes    map[string]string
	schemaResourceHashes map[string]string
	schemaServiceHashes  map[string]string
}

func (s *Syncer) DIInit() {
	s.appConfigNamesById = map[int]string{}
	s.portalSiteNamesById = map[int]string{}
	s.portalRuleNamesById = map[int]string{}
	s.portalCertNamesById = map[int]string{}
	s.schemaActorHashes = map[string]string{}
	s.schemaResourceHashes = map[string]string{}
	s.schemaServiceHashes = map[string]string{}
}

func (s *Syncer) SyncAppConfig(item *core.AppConfig) {
	s.removeRenamedKey(s.appConfigNamesById, item.Id, item.Name, redised.FormatConfigKey)
	s.RedisServer.SetAndNotify(redised.FormatConfigKey(item.Name), vcode.MustMarshalJsonS(ToRedisedAppConfig(item)))
	s.saveNameById(s.appConfigNamesById, item.Id, item.Name)
}

func (s *Syncer) RemoveAppConfig(item *core.AppConfig) {
	s.RedisServer.DeleteAndNotify(redised.FormatConfigKey(item.Name))
	delete(s.appConfigNamesById, item.Id)
}

func (s *Syncer) SyncRpcServiceRegistration(reg redised.RpcServiceRegistration) {
	s.RedisServer.SetAndNotify(
		redised.FormatRpcServiceRegistrationKey(reg.ServiceName, reg.AppName, reg.AppInstanceId),
		vcode.MustMarshalJsonS(reg),
	)
}

func (s *Syncer) SyncWebRegistration(webName string, reg redised.WebRegistration) {
	s.RedisServer.SetAndNotify(
		redised.FormatWebRegistrationKey(webName, reg.AppName, reg.AppInstanceId),
		vcode.MustMarshalJsonS(reg),
	)
}

func (s *Syncer) removeRenamedKey(namesById map[int]string, id int, name string, formatKey func(string) string) {
	if oldName, ok := namesById[id]; ok && oldName != name {
		s.RedisServer.DeleteAndNotify(formatKey(oldName))
	}
}

func (s *Syncer) saveNameById(namesById map[int]string, id int, name string) {
	namesById[id] = name
}

func ToRedisedAppConfig(item *core.AppConfig) *redised.ConfigValue {
	return &redised.ConfigValue{
		Name:  item.Name,
		Value: json.RawMessage(item.Value),
	}
}
