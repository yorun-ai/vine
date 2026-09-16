package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// SyncPortalEntry records the entry Hub publishes rules for. An entry owns the
// access of its rules, so its enable switch decides whether Portal sees them.
func (s *Syncer) SyncPortalEntry(entry *core.PortalEntry) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	stored := *entry
	s.portalEntriesById[entry.Id] = &stored
	for _, rule := range s.portalRulesById {
		if rule.EntryId != entry.Id {
			continue
		}
		s.publishPortalRuleLocked(rule, s.portalSiteOfRuleLocked(rule))
	}
}

func (s *Syncer) RemovePortalEntry(entry *core.PortalEntry) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	delete(s.portalEntriesById, entry.Id)
	for _, rule := range s.portalRulesById {
		if rule.EntryId != entry.Id {
			continue
		}
		// Hub publishes a rule when it cannot resolve the entry it belongs to, so
		// removing the entry leaves the published rule in place.
		s.publishPortalRuleLocked(rule, s.portalSiteOfRuleLocked(rule))
	}
}

// SyncPortalSite publishes a complete portal site, including the services it
// derives from the schemas registered for its actor.
func (s *Syncer) SyncPortalSite(site *core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.cachePortalSiteLocked(site)
	if !site.Enabled {
		// Portal keeps reading what Hub already published for the site, and a
		// disabled site stops serving its rules.
		s.WatchServer.DeleteAndNotify(watched.FormatPortalSiteKey(site.Name))
		delete(s.portalSiteNamesById, site.Id)
		for _, rule := range s.portalRulesById {
			if rule.RouteSiteName == site.Name {
				s.publishPortalRuleLocked(rule, site)
			}
		}
		return
	}
	s.removeRenamedKeyLocked(s.portalSiteNamesById, site.Id, site.Name, watched.FormatPortalSiteKey)
	s.WatchServer.SetAndNotify(watched.FormatPortalSiteKey(site.Name), vcode.MustMarshalJsonS(toWatchedPortalSite(site)))
	s.saveNameByIdLocked(s.portalSiteNamesById, site.Id, site.Name)
	for _, rule := range s.portalRulesById {
		if rule.RouteSiteName == site.Name {
			s.publishPortalRuleLocked(rule, site)
		}
	}
}

func (s *Syncer) RemovePortalSite(site *core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removePortalSiteLocked(site)
	s.WatchServer.DeleteAndNotify(watched.FormatPortalSiteKey(site.Name))
	delete(s.portalSiteNamesById, site.Id)
	for _, rule := range s.portalRulesById {
		if rule.RouteSiteName == site.Name {
			s.publishPortalRuleLocked(rule, nil)
		}
	}
}

func (s *Syncer) SyncPortalRule(rule *core.PortalRule, sites ...*core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removeRenamedKeyLocked(s.portalRuleNamesById, rule.Id, rule.Name, watched.FormatPortalRuleKey)
	var site *core.PortalSite
	if len(sites) > 0 {
		site = sites[0]
	}
	s.publishPortalRuleLocked(rule, site)
	s.saveNameByIdLocked(s.portalRuleNamesById, rule.Id, rule.Name)
	s.portalRulesById[rule.Id] = clonePortalRule(rule)
}

func (s *Syncer) RemovePortalRule(rule *core.PortalRule) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatPortalRuleKey(rule.Name))
	delete(s.portalRuleNamesById, rule.Id)
	delete(s.portalRulesById, rule.Id)
}

func clonePortalRule(rule *core.PortalRule) *core.PortalRule {
	copy := *rule
	return &copy
}

func (s *Syncer) SyncPortalCert(cert *core.PortalCert) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	if !cert.Enabled {
		// A disabled certificate is not part of the Portal TLS configuration.
		s.WatchServer.DeleteAndNotify(watched.FormatPortalCertKey(cert.Name))
		delete(s.portalCertNamesById, cert.Id)
		return
	}
	s.removeRenamedKeyLocked(s.portalCertNamesById, cert.Id, cert.Name, watched.FormatPortalCertKey)
	s.WatchServer.SetAndNotify(watched.FormatPortalCertKey(cert.Name), vcode.MustMarshalJsonS(ToWatchedPortalCert(cert)))
	s.saveNameByIdLocked(s.portalCertNamesById, cert.Id, cert.Name)
}

func (s *Syncer) RemovePortalCert(cert *core.PortalCert) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatPortalCertKey(cert.Name))
	delete(s.portalCertNamesById, cert.Id)
}

// publishPortalRuleLocked publishes one rule for Portal, or removes it when Hub
// does not publish it: a disabled rule, a rule of a disabled entry, or a SITE
// rule whose site is disabled.
func (s *Syncer) publishPortalRuleLocked(rule *core.PortalRule, site *core.PortalSite) {
	if !s.publishesPortalRuleLocked(rule, site) {
		s.WatchServer.DeleteAndNotify(watched.FormatPortalRuleKey(rule.Name))
		return
	}
	s.WatchServer.SetAndNotify(watched.FormatPortalRuleKey(rule.Name), vcode.MustMarshalJsonS(s.toWatchedPortalRule(rule, site)))
}

func (s *Syncer) publishesPortalRuleLocked(rule *core.PortalRule, site *core.PortalSite) bool {
	if !rule.Enabled {
		return false
	}
	// Hub publishes a rule when it cannot resolve the entry it belongs to, so a
	// partial startup never hides configuration from Portal.
	if entry, ok := s.portalEntriesById[rule.EntryId]; ok && !entry.Enabled {
		return false
	}
	if rule.RouteType == core.PortalRuleRouteTypeSite && site != nil && !site.Enabled {
		return false
	}
	return true
}

// portalSiteOfRuleLocked returns the site a rule targets, from the sites Hub has
// published.
func (s *Syncer) portalSiteOfRuleLocked(rule *core.PortalRule) *core.PortalSite {
	if rule.RouteType != core.PortalRuleRouteTypeSite {
		return nil
	}
	return s.portalSitesByName[rule.RouteSiteName]
}

func toWatchedPortalSite(site *core.PortalSite) *watched.PortalSite {
	ret := &watched.PortalSite{
		Name: site.Name,
		Type: string(site.Type),
		ActorVia: watched.PortalActorVia{
			ActorSkelName: site.ActorSkelName,
			ActorVia:      site.ActorVia,
		},
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
		ret.RpcgwConfig = &watched.PortalRpcgwConfig{Services: services}
	}
	if site.Type == core.PortalSiteTypeWEBGW {
		ret.WebgwConfig = &watched.PortalWebgwConfig{WebName: site.WebName}
		ret.WebgwConfig.MountPath = site.WebMountPath
	}
	return ret
}

func (s *Syncer) toWatchedPortalRule(rule *core.PortalRule, sites ...*core.PortalSite) *watched.PortalRule {
	ret := ToWatchedPortalRule(rule)
	if len(sites) > 0 {
		ret.ResolvedMatchPathPrefix, ret.ResolvedRoutePathPrefix = core.ResolvePortalRulePaths(rule, sites[0])
	}
	return ret
}

func ToWatchedPortalRule(rule *core.PortalRule) *watched.PortalRule {
	return &watched.PortalRule{
		Name:                    rule.Name,
		MatchScheme:             rule.MatchScheme,
		MatchHost:               rule.MatchHost,
		MatchPort:               rule.MatchPort,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		ResolvedMatchPathPrefix: rule.MatchPathPrefix,
		ResolvedRoutePathPrefix: rule.RoutePathPrefix,
	}
}

func ToWatchedPortalCert(cert *core.PortalCert) *watched.PortalCert {
	return &watched.PortalCert{
		Name:             cert.Name,
		Issuer:           cert.Issuer,
		PublicKeyBase64:  cert.PublicKeyBase64,
		PrivateKeyBase64: cert.PrivateKeyBase64,
		ValidFrom:        cert.ValidFrom,
		ValidTo:          cert.ValidTo,
	}
}
func (s *Syncer) cachePortalSiteLocked(site *core.PortalSite) {
	stored := *site
	s.portalSitesByName[site.Name] = &stored
}

func (s *Syncer) removePortalSiteLocked(site *core.PortalSite) {
	delete(s.portalSitesByName, site.Name)
}
