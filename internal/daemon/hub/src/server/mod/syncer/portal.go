package syncer

import (
	"strings"

	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// SyncPortalSite publishes a complete portal site, including the services it
// derives from the schemas registered for its actor.
func (s *Syncer) SyncPortalSite(site *core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removeRenamedKeyLocked(s.portalSiteNamesById, site.Id, site.Name, watched.FormatPortalSiteKey)
	s.WatchServer.SetAndNotify(watched.FormatPortalSiteKey(site.Name), vcode.MustMarshalJsonS(toWatchedPortalSite(site)))
	s.saveNameByIdLocked(s.portalSiteNamesById, site.Id, site.Name)
}

func (s *Syncer) RemovePortalSite(site *core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatPortalSiteKey(site.Name))
	delete(s.portalSiteNamesById, site.Id)
}

func (s *Syncer) SyncPortalRule(rule *core.PortalRule, sites ...*core.PortalSite) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.removeRenamedKeyLocked(s.portalRuleNamesById, rule.Id, rule.Name, watched.FormatPortalRuleKey)
	s.WatchServer.SetAndNotify(watched.FormatPortalRuleKey(rule.Name), vcode.MustMarshalJsonS(s.toWatchedPortalRule(rule, sites...)))
	s.saveNameByIdLocked(s.portalRuleNamesById, rule.Id, rule.Name)
}

func (s *Syncer) RemovePortalRule(rule *core.PortalRule) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

	s.WatchServer.DeleteAndNotify(watched.FormatPortalRuleKey(rule.Name))
	delete(s.portalRuleNamesById, rule.Id)
}

func (s *Syncer) SyncPortalCert(cert *core.PortalCert) {
	s.namesMutex.Lock()
	defer s.namesMutex.Unlock()

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
		if !site.BuiltIn {
			ret.WebgwConfig.MountPath = site.WebMountPath
		}
	}
	return ret
}

func (s *Syncer) toWatchedPortalRule(rule *core.PortalRule, sites ...*core.PortalSite) *watched.PortalRule {
	ret := ToWatchedPortalRule(rule)
	if len(sites) > 0 {
		site := sites[0]
		if site == nil || site.WebMountPath == "" || rule.RouteType != string(core.PortalRuleRouteTypeSite) {
			return ret
		}
		mountPath := strings.TrimRight(site.WebMountPath, "/")
		ret.ResolvedMatchPathPrefix = mountPath
		ret.ResolvedRoutePathPrefix = mountPath
		if mountPath == "" {
			ret.ResolvedMatchPathPrefix = "/"
		}
	}
	return ret
}

func ToWatchedPortalRule(rule *core.PortalRule) *watched.PortalRule {
	return &watched.PortalRule{
		Name:                    rule.Name,
		MatchScheme:             rule.MatchScheme,
		MatchHost:               rule.MatchHost,
		MatchPort:               rule.MatchPort,
		MatchPathPrefix:         rule.MatchPathPrefix,
		RouteType:               rule.RouteType,
		RouteSiteName:           rule.RouteSiteName,
		RouteRedirectionPattern: rule.RouteRedirectionPattern,
		RoutePathPrefix:         rule.RoutePathPrefix,
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
