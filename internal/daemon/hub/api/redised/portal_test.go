package redised

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPortalRuleKeys(t *testing.T) {
	assert.Equal(t, "portal:rule:demo-entry", FormatPortalRuleKey("demo-entry"))
}

func TestPortalSiteKeys(t *testing.T) {
	assert.Equal(t, "portal:site:demo.user.AdminActor-client-rpc", FormatPortalSiteKey("demo.user.AdminActor-client-rpc"))
}

func TestPortalSiteJSON(t *testing.T) {
	data, err := json.Marshal(PortalSite{
		Name: "demo.user.AdminActor-client-rpc",
		Type: "RPCGW",
		ActorVia: PortalActorVia{
			ActorSkelName: "demo.user.AdminActor",
			ActorVia:      "client",
		},
		Cors: PortalCors{
			Mode:           PortalCorsModeStrict,
			AllowedOrigins: []string{"https://console.example.com"},
		},
		RpcgwConfig: &PortalRpcgwConfig{
			Services: []PortalRpcgwService{
				{SkelName: "demo.user.UserService"},
				{SkelName: "demo.order.OrderService"},
			},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "demo.user.AdminActor-client-rpc",
		"type": "RPCGW",
		"actorVia": {
			"actorSkelName": "demo.user.AdminActor",
			"actorVia": "client"
		},
		"cors": {
			"mode": "STRICT",
			"allowedOrigins": ["https://console.example.com"]
		},
		"rpcgwConfig": {
			"services": [
				{"skelName": "demo.user.UserService"},
				{"skelName": "demo.order.OrderService"}
			]
		}
	}`, string(data))
}

func TestPortalCertKeys(t *testing.T) {
	assert.Equal(t, "portal:cert:demo-cert", FormatPortalCertKey("demo-cert"))
}
