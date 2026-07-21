package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func (s *Syncer) SyncPortalSiteWithRpcgwServices(site *core.PortalSite, rpcgwServices []string) {
	s.removeRenamedKey(s.portalSiteNamesById, site.Id, site.Name, redised.FormatPortalSiteKey)
	s.RedisServer.SetAndNotify(redised.FormatPortalSiteKey(site.Name), vcode.MustMarshalJsonS(toRedisedPortalSite(site, rpcgwServices)))
	s.saveNameById(s.portalSiteNamesById, site.Id, site.Name)
}

func (s *Syncer) RemovePortalSite(site *core.PortalSite) {
	s.RedisServer.DeleteAndNotify(redised.FormatPortalSiteKey(site.Name))
	delete(s.portalSiteNamesById, site.Id)
}

func (s *Syncer) SyncPortalRule(rule *core.PortalRule) {
	s.removeRenamedKey(s.portalRuleNamesById, rule.Id, rule.Name, redised.FormatPortalRuleKey)
	s.RedisServer.SetAndNotify(redised.FormatPortalRuleKey(rule.Name), vcode.MustMarshalJsonS(ToRedisedPortalRule(rule)))
	s.saveNameById(s.portalRuleNamesById, rule.Id, rule.Name)
}

func (s *Syncer) RemovePortalRule(rule *core.PortalRule) {
	s.RedisServer.DeleteAndNotify(redised.FormatPortalRuleKey(rule.Name))
	delete(s.portalRuleNamesById, rule.Id)
}

func (s *Syncer) SyncPortalCert(cert *core.PortalCert) {
	s.removeRenamedKey(s.portalCertNamesById, cert.Id, cert.Name, redised.FormatPortalCertKey)
	s.RedisServer.SetAndNotify(redised.FormatPortalCertKey(cert.Name), vcode.MustMarshalJsonS(ToRedisedPortalCert(cert)))
	s.saveNameById(s.portalCertNamesById, cert.Id, cert.Name)
}

func (s *Syncer) RemovePortalCert(cert *core.PortalCert) {
	s.RedisServer.DeleteAndNotify(redised.FormatPortalCertKey(cert.Name))
	delete(s.portalCertNamesById, cert.Id)
}

func toRedisedPortalSite(site *core.PortalSite, rpcgwServices []string) *redised.PortalSite {
	ret := &redised.PortalSite{
		Name: site.Name,
		Type: string(site.Type),
		ActorVia: redised.PortalActorVia{
			ActorSkelName: site.ActorSkelName,
			ActorVia:      site.ActorVia,
		},
		Cors: redised.PortalCors{
			Mode:           redised.PortalCorsMode(site.Cors.Mode),
			AllowedOrigins: append([]string{}, site.Cors.AllowedOrigins...),
		},
	}
	if site.Type == core.PortalSiteTypeRPCGW {
		services := make([]redised.PortalRpcgwService, 0, len(rpcgwServices))
		for _, serviceName := range rpcgwServices {
			services = append(services, redised.PortalRpcgwService{SkelName: serviceName})
		}
		ret.RpcgwConfig = &redised.PortalRpcgwConfig{Services: services}
	}
	if site.Type == core.PortalSiteTypeWEBGW {
		ret.WebgwConfig = &redised.PortalWebgwConfig{WebName: site.WebName}
	}
	return ret
}

func ToRedisedPortalRule(rule *core.PortalRule) *redised.PortalRule {
	return &redised.PortalRule{
		Name:               rule.Name,
		Scheme:             rule.Scheme,
		Host:               rule.Host,
		Port:               rule.Port,
		PathPrefix:         rule.PathPrefix,
		TargetType:         rule.TargetType,
		SiteName:           rule.SiteName,
		RedirectionPattern: rule.RedirectionPattern,
	}
}

func ToRedisedPortalCert(cert *core.PortalCert) *redised.PortalCert {
	return &redised.PortalCert{
		Name:             cert.Name,
		Issuer:           cert.Issuer,
		PublicKeyBase64:  cert.PublicKeyBase64,
		PrivateKeyBase64: cert.PrivateKeyBase64,
		ValidFrom:        cert.ValidFrom,
		ValidTo:          cert.ValidTo,
	}
}
