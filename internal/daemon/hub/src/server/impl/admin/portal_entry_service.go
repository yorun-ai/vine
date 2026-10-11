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
	validatePortalEntryProtocol(update.Protocol)
	entry := s.PortalEntryCore.Update(id, core.PortalEntryUpdate{
		Protocol: update.Protocol, Http: toCoreHTTPConfig(update.Http),
		Name:      update.Name,
		ListenIPs: update.ListenIPs,
		Enabled:   update.Enabled,
	})
	return s.toServerPortalEntry(entry)
}

func (s *PortalEntryApiServiceServerImpl) Create(creation skeled.PortalEntryCreation) skeled.PortalEntry {
	validatePortalEntryProtocol(&creation.Protocol)
	config := core.DefaultPortalEntryHTTP()
	if creation.Http != nil {
		config = *toCoreHTTPConfig(creation.Http)
	}
	entry := s.PortalEntryCore.Create(core.PortalEntryCreation{
		Protocol: creation.Protocol, Http: &config,
		Name:      creation.Name,
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
		Protocol:      entry.Protocol,
		EffectiveHttp: toServerEffectiveHTTP(*entry.EffectiveHTTP),
		Http:          skeled.PortalEntryHttp{Host: entry.Http.Host, Http: skeled.PortalSwitchMode(entry.Http.Http), HttpPort: entry.Http.HttpPort, Https: skeled.PortalSwitchMode(entry.Http.Https), HttpsPort: entry.Http.HttpsPort, Autohttps: skeled.PortalSwitchMode(entry.Http.AutoHTTPS)},
		Id:            entry.Id,
		Name:          entry.Name,
		ListenIPs:     entry.ListenIPs,
		Enabled:       entry.Enabled,
		Rules:         rules,
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

func validatePortalEntryProtocol(protocol *string) {
	ex.PanicNewIfNot(protocol == nil || *protocol == "http", ex.OperationFailed, "unknown portal entry protocol")
}
func toCoreHTTPConfig(h *skeled.PortalEntryHttp) *core.PortalEntryHTTP {
	if h == nil {
		return nil
	}
	return &core.PortalEntryHTTP{Host: h.Host, Http: core.PortalSwitchMode(h.Http), HttpPort: h.HttpPort, Https: core.PortalSwitchMode(h.Https), HttpsPort: h.HttpsPort, AutoHTTPS: core.PortalSwitchMode(h.Autohttps)}
}

func toServerEffectiveHTTP(h core.PortalEntryEffectiveHTTP) skeled.PortalEntryEffectiveHttp {
	return skeled.PortalEntryEffectiveHttp{Host: h.Host, HttpEnabled: h.HttpEnabled, HttpPort: h.HttpPort, HttpsEnabled: h.HttpsEnabled, HttpsPort: h.HttpsPort, AutohttpsEnabled: h.AutoHTTPSEnabled, HttpsReason: h.HTTPSReason}
}
