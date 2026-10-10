package admin

import (
	"go.yorun.ai/vine/internal/core/ex"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type PortalEntryApiServiceServerImpl struct {
	skeled.DefaultPortalEntryApiServiceServer

	PortalEntryCore *core.PortalEntryCore `inject:""`
}

func (s *PortalEntryApiServiceServerImpl) List() []skeled.PortalEntry {
	entries := s.PortalEntryCore.List()
	ret := make([]skeled.PortalEntry, 0, len(entries))
	for _, entry := range entries {
		ret = append(ret, s.toServerPortalEntry(entry))
	}
	return ret
}

func (s *PortalEntryApiServiceServerImpl) Update(id int, update skeled.PortalEntryUpdate) skeled.PortalEntry {
	validatePortalEntryVocabulary(update.Protocol, update.Http, update.Scheme, update.Port)
	entry := s.PortalEntryCore.Update(id, core.PortalEntryUpdate{
		Protocol: update.Protocol, Http: toCoreHTTPUpdate(update.Http),
		Name:      update.Name,
		Scheme:    update.Scheme,
		Host:      update.Host,
		Port:      update.Port,
		ListenIPs: update.ListenIPs,
		Enabled:   update.Enabled,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Create(creation skeled.PortalEntryCreation) skeled.PortalEntry {
	validatePortalEntryVocabulary(creation.Protocol, creation.Http, creation.Scheme, creation.Port)
	var config *core.PortalEntryHTTP
	protocol, scheme, port := "", "", 0
	if creation.Protocol != nil {
		protocol = *creation.Protocol
		value := core.DefaultPortalEntryHTTP()
		if creation.Http != nil {
			value = toCoreHTTPUpdate(creation.Http).Apply(value)
		}
		config = &value
	}
	if creation.Scheme != nil {
		scheme = *creation.Scheme
	}
	if creation.Port != nil {
		port = *creation.Port
	}
	entry := s.PortalEntryCore.Create(core.PortalEntryCreation{
		Protocol: protocol, Http: config,
		Name:      creation.Name,
		Scheme:    scheme,
		Host:      creation.Host,
		Port:      port,
		ListenIPs: creation.ListenIPs,
		Enabled:   creation.Enabled,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Remove(id int) {
	s.PortalEntryCore.Remove(id)
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntry(entry core.PortalEntryView) skeled.PortalEntry {
	entry.PortalEntry = core.NormalizePortalEntry(entry.PortalEntry)
	rules := make([]skeled.PortalEntryRule, 0, len(entry.Rules))
	for _, rule := range entry.Rules {
		rules = append(rules, s.toServerPortalEntryRule(entry.PortalEntry, rule))
	}
	return skeled.PortalEntry{
		Protocol:  entry.Protocol,
		Http:      skeled.PortalEntryHttp{HttpEnabled: entry.Http.HttpEnabled, HttpPort: entry.Http.HttpPort, HttpsEnabled: entry.Http.HttpsEnabled, HttpsPort: entry.Http.HttpsPort, AutoHttps: entry.Http.AutoHTTPS},
		Id:        entry.Id,
		Name:      entry.Name,
		Scheme:    entry.Scheme,
		Host:      entry.Host,
		Port:      entry.Port,
		ListenIPs: entry.ListenIPs,
		Enabled:   entry.Enabled,
		Rules:     rules,
	}
}

func (s *PortalEntryApiServiceServerImpl) toServerPortalEntryRule(entry core.PortalEntry, rule core.PortalEntryRule) skeled.PortalEntryRule {
	var site *skeled.PortalSiteListItem
	if rule.Site != nil {
		value := toServerPortalSiteListItem(rule.Site)
		site = &value
	}
	return skeled.PortalEntryRule{
		Rule: toServerPortalRuleListItem(&entry, rule.Rule, nil, rule.Site),
		Site: site,
	}
}

func validatePortalEntryVocabulary(protocol *string, config *skeled.PortalEntryHttpUpdate, scheme *string, port *int) {
	ex.PanicNewIfNot(protocol == nil || (scheme == nil && port == nil), ex.OperationFailed, "protocol cannot be combined with scheme or port")
	ex.PanicNewIfNot(config == nil || protocol != nil, ex.OperationFailed, "http requires protocol")
	ex.PanicNewIfNot(protocol == nil || *protocol == "http", ex.OperationFailed, "unknown portal entry protocol")
}
func toCoreHTTPUpdate(h *skeled.PortalEntryHttpUpdate) *core.PortalEntryHTTPUpdate {
	if h == nil {
		return nil
	}
	return &core.PortalEntryHTTPUpdate{HttpEnabled: h.HttpEnabled, HttpPort: h.HttpPort, HttpsEnabled: h.HttpsEnabled, HttpsPort: h.HttpsPort, AutoHTTPS: h.AutoHttps}
}
