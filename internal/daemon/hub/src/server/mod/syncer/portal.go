package syncer

import (
	"slices"

	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/watchserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// SyncPortalEntry records the entry Hub publishes rules for. An entry owns the
// access of its rules, so its enable switch decides whether Portal sees them.
func (s *Syncer) SyncPortalEntry(entry *core.PortalEntry) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	stored := core.NormalizePortalEntry(*entry)
	stored.EffectiveHTTP = nil
	previous := s.portalEntriesById[entry.Id]
	if previous != nil && previous.Name != entry.Name {
		batch.Delete(watched.FormatPortalEntryKey(previous.Name))
	}
	s.publishPortalEntryLocked(batch, &stored)
	stored.ListenIPs = slices.Clone(stored.ListenIPs)
	s.portalEntriesById[entry.Id] = &stored
	if previous == nil || previous.Enabled != stored.Enabled || previous.Name != stored.Name ||
		previous.Protocol != stored.Protocol || !previous.Http.Equal(*stored.Http) {
		s.updateEntryRules(batch, entry.Id)
	}
	batch.Notify()
}

func (s *Syncer) RemovePortalEntry(entry *core.PortalEntry) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	batch.Delete(watched.FormatPortalEntryKey(entry.Name))
	delete(s.portalEntriesById, entry.Id)
	s.updateEntryRules(batch, entry.Id)
	batch.Notify()
}

func (s *Syncer) updateEntryRules(batch *watchserver.NotifyBatch, entryID int) {
	changed := make(map[string]struct{})
	s.reindexRules(s.rules.byEntry[entryID], changed)
	s.publishRuleGroups(batch, changed)
}

// SyncPortalSite publishes the site and any changes to its rules in one revision.
func (s *Syncer) SyncPortalSite(site *core.PortalSite) {
	s.SyncPortalSites([]*core.PortalSite{site})
}

// SyncPortalSites applies all mount paths from one descriptor refresh together.
func (s *Syncer) SyncPortalSites(sites []*core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	changedSites := make(map[string]struct{})
	// Resolve old names first, so a name can move to another site in this refresh.
	for _, site := range sites {
		oldName := s.portalSiteNamesById[site.Id]
		if oldName != "" && oldName != site.Name {
			delete(s.portalSitesByName, oldName)
			changedSites[oldName] = struct{}{}
		}
	}
	for _, site := range sites {
		previous := s.portalSitesByName[site.Name]
		s.portalSiteNamesById[site.Id] = site.Name
		s.portalSitesByName[site.Name] = new(*site)
		key := watched.FormatPortalSiteKey(site.Name)
		if site.Enabled {
			batch.Set(key, vcode.MustMarshalJsonS(toWatchedPortalSite(site)))
		} else {
			batch.Delete(key)
		}
		if previous == nil || previous.Enabled != site.Enabled ||
			previous.Type != site.Type || previous.WebMountPath != site.WebMountPath {
			changedSites[site.Name] = struct{}{}
		}
	}
	changed := make(map[string]struct{})
	for name := range changedSites {
		if s.portalSitesByName[name] == nil {
			batch.Delete(watched.FormatPortalSiteKey(name))
		}
		s.reindexRules(s.rules.bySite[name], changed)
	}
	s.publishRuleGroups(batch, changed)
	batch.Notify()
}

func (s *Syncer) RemovePortalSite(site *core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	batch.Delete(watched.FormatPortalSiteKey(site.Name))
	delete(s.portalSitesByName, site.Name)
	delete(s.portalSiteNamesById, site.Id)
	changed := make(map[string]struct{})
	s.reindexRules(s.rules.bySite[site.Name], changed)
	s.publishRuleGroups(batch, changed)
	batch.Notify()
}

func (s *Syncer) SyncPortalCert(cert *core.PortalCert) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removeRenamedKeyLocked(s.portalCertNamesById, cert.Id, cert.Name, watched.FormatPortalCertKey)
	if !cert.Enabled {
		// A disabled certificate is not part of the Portal TLS configuration.
		s.WatchServer.DeleteAndNotify(watched.FormatPortalCertKey(cert.Name))
		delete(s.portalCertNamesById, cert.Id)
		delete(s.portalCertsById, cert.Id)
		s.republishPortalEntriesLocked()
		return
	}
	s.WatchServer.SetAndNotify(watched.FormatPortalCertKey(cert.Name), vcode.MustMarshalJsonS(ToWatchedPortalCert(cert)))
	s.saveNameByIdLocked(s.portalCertNamesById, cert.Id, cert.Name)
	stored := *cert
	stored.Domains = slices.Clone(cert.Domains)
	s.portalCertsById[cert.Id] = &stored
	s.republishPortalEntriesLocked()
}

func (s *Syncer) RemovePortalCert(cert *core.PortalCert) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatPortalCertKey(cert.Name))
	delete(s.portalCertNamesById, cert.Id)
	delete(s.portalCertsById, cert.Id)
	s.republishPortalEntriesLocked()
}

// portalSiteOfRuleLocked returns the site a rule targets, including disabled sites.
func (s *Syncer) portalSiteOfRuleLocked(rule *core.PortalRule) *core.PortalSite {
	if rule.RouteType != core.PortalRuleRouteTypeSite {
		return nil
	}
	return s.portalSitesByName[rule.RouteSiteName]
}

func toWatchedPortalSite(site *core.PortalSite) *watched.PortalSite {
	ret := &watched.PortalSite{
		Name:          site.Name,
		Type:          string(site.Type),
		ActorSkelName: site.ActorSkelName,
		WebTimeout:    site.WebTimeout,
		Cors: watched.PortalCors{
			Mode:           watched.PortalCorsMode(site.Cors.Mode),
			AllowedOrigins: append([]string{}, site.Cors.AllowedOrigins...),
		},
	}
	if site.Type == core.PortalSiteTypeRPCGW {
		services := make([]watched.PortalRpcgwService, 0, len(site.RpcgwServices))
		for _, serviceName := range site.RpcgwServices {
			services = append(services, watched.PortalRpcgwService{SkelName: serviceName})
		}
		ret.RpcgwConfig = &watched.PortalRpcgwConfig{
			InspectEnabled: site.InspectEnabled,
			McpEnabled:     site.McpEnabled,
			Services:       services,
		}
	}
	if site.Type == core.PortalSiteTypeWEBGW {
		ret.WebgwConfig = &watched.PortalWebgwConfig{WebName: site.WebName}
		ret.WebgwConfig.MountPath = site.WebMountPath
	}
	return ret
}

func (s *Syncer) toWatchedPortalRule(rule *core.PortalRule, site *core.PortalSite) *watched.PortalRule {
	ret := ToWatchedPortalRule(rule, s.portalEntriesById[rule.EntryId])
	ret.ResolvedMatchPathPrefix, ret.ResolvedRoutePathPrefix = core.ResolvePortalRulePaths(rule, site)
	return ret
}

// ToWatchedPortalRule renders a rule with its entry reference and resolved paths.
func ToWatchedPortalRule(rule *core.PortalRule, entry *core.PortalEntry) *watched.PortalRule {
	ret := &watched.PortalRule{
		EntryName:               entry.Name,
		Name:                    rule.Name,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		ResolvedMatchPathPrefix: rule.MatchPathPrefix,
		ResolvedRoutePathPrefix: rule.RoutePathPrefix,
	}
	return ret
}

func ToWatchedPortalCert(cert *core.PortalCert) *watched.PortalCert {
	return &watched.PortalCert{
		Name:        cert.Name,
		Issuer:      cert.Issuer,
		Certificate: cert.Certificate,
		PrivateKey:  cert.PrivateKey,
		ValidFrom:   cert.ValidFrom,
		ValidTo:     cert.ValidTo,
	}
}

func (s *Syncer) publishPortalEntryLocked(batch *watchserver.NotifyBatch, entry *core.PortalEntry) {
	if !entry.Enabled {
		batch.Delete(watched.FormatPortalEntryKey(entry.Name))
		return
	}
	certs := make([]*core.PortalCert, 0, len(s.portalCertsById))
	for _, cert := range s.portalCertsById {
		certs = append(certs, cert)
	}
	h := entry.Http.Effective(core.HasPortalCertificate(entry.Http.Host, certs))
	batch.Set(watched.FormatPortalEntryKey(entry.Name), vcode.MustMarshalJsonS(watched.PortalEntry{Name: entry.Name, Protocol: entry.Protocol, ListenIPs: entry.ListenIPs, EffectiveHTTP: watched.PortalEntryHTTP{Host: h.Host, HttpEnabled: h.HttpEnabled, HttpPort: h.HttpPort, HttpsEnabled: h.HttpsEnabled, HttpsPort: h.HttpsPort, AutoHTTPSEnabled: h.AutoHTTPSEnabled}}))
}

func (s *Syncer) republishPortalEntriesLocked() {
	batch := s.WatchServer.NotifyBatch()
	for _, entry := range s.portalEntriesById {
		s.publishPortalEntryLocked(batch, entry)
	}
	batch.Notify()
}
