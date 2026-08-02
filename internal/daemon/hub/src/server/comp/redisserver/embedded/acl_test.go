package embedded

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/hub/api/redis"
)

func TestLinkACLAllowsOnlyConfigurationAndRpcDiscovery(t *testing.T) {
	for name, command := range map[string][]string{
		"ping":                    {"PING"},
		"revision":                {"GET", redis.RevisionKey},
		"configuration":           {"GET", "config:demo.Feature"},
		"rpc registration":        {"GET", "rpc:demo.Service:endpoint:demo.app:instance-1"},
		"rpc namespace scan":      {"SCAN", "0", "MATCH", "rpc:*"},
		"rpc scan":                {"SCAN", "0", "MATCH", "rpc:demo.Service:endpoint:*", "COUNT", "1000"},
		"configuration subscribe": {"SUBSCRIBE", "config:demo.Feature"},
		"rpc psubscribe":          {"PSUBSCRIBE", "rpc:demo.Service:endpoint:*"},
		"unsubscribe all":         {"UNSUBSCRIBE"},
		"punsubscribe rpc":        {"PUNSUBSCRIBE", "rpc:demo.Service:endpoint:*"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, canRunTestCommand(userRoleLink, command))
		})
	}

	for name, command := range map[string][]string{
		"anonymous command":            {"GET", "config:demo.Feature"},
		"write":                        {"SET", "config:demo.Feature", "value"},
		"ttl":                          {"TTL", "config:demo.Feature"},
		"empty configuration key":      {"GET", "config:"},
		"portal certificate":           {"GET", "portal:cert:production"},
		"portal schema":                {"GET", "schema:service:demo.Service"},
		"web registration":             {"GET", "web:admin@demo.app:endpoint:demo.app:instance-1"},
		"application status":           {"GET", "app:demo.app:status:instance-1"},
		"scan all":                     {"SCAN", "0"},
		"scan configuration":           {"SCAN", "0", "MATCH", "config:*"},
		"subscribe portal certificate": {"SUBSCRIBE", "portal:cert:production"},
		"subscribe mixed channels":     {"SUBSCRIBE", "config:demo.Feature", "portal:cert:production"},
		"psubscribe all":               {"PSUBSCRIBE", "*"},
		"psubscribe portal":            {"PSUBSCRIBE", "portal:cert:*"},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			role := userRoleLink
			if name == "anonymous command" {
				role = userRoleNone
			}
			assert.False(t, canRunTestCommand(role, command))
		})
	}
}

func TestPortalACLAllowsOnlyPortalRuntimeState(t *testing.T) {
	for name, command := range map[string][]string{
		"ping":                    {"PING"},
		"revision":                {"GET", redis.RevisionKey},
		"portal rule":             {"GET", "portal:rule:public"},
		"portal site":             {"GET", "portal:site:public"},
		"portal certificate":      {"GET", "portal:cert:production"},
		"actor schema":            {"GET", "schema:actor:demo.Actor"},
		"service schema":          {"GET", "schema:service:demo.Service"},
		"resource schema":         {"GET", "schema:resource:demo.Resource"},
		"rpc registration":        {"GET", "rpc:demo.Service:endpoint:demo.app:instance-1"},
		"web registration":        {"GET", "web:admin@demo.app:endpoint:demo.app:instance-1"},
		"portal certificate scan": {"SCAN", "0", "MATCH", "portal:cert:*"},
		"service schema scan":     {"SCAN", "0", "MATCH", "schema:service:*"},
		"rpc scan":                {"SCAN", "0", "MATCH", "rpc:demo.Service:endpoint:*"},
		"web scan":                {"SCAN", "0", "MATCH", "web:admin@demo.app:endpoint:*"},
		"portal psubscribe":       {"PSUBSCRIBE", "portal:rule:*"},
		"rpc psubscribe":          {"PSUBSCRIBE", "rpc:demo.Service:endpoint:*"},
		"punsubscribe all":        {"PUNSUBSCRIBE"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, canRunTestCommand(userRolePortal, command))
		})
	}

	for name, command := range map[string][]string{
		"write":                     {"DEL", "portal:cert:production"},
		"configuration":             {"GET", "config:demo.Feature"},
		"application status":        {"GET", "app:demo.app:status:instance-1"},
		"empty portal key":          {"GET", "portal:cert:"},
		"scan all":                  {"SCAN", "0"},
		"scan broad portal pattern": {"SCAN", "0", "MATCH", "portal:*"},
		"configuration subscribe":   {"SUBSCRIBE", "config:demo.Feature"},
		"psubscribe all":            {"PSUBSCRIBE", "*"},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			assert.False(t, canRunTestCommand(userRolePortal, command))
		})
	}
}

func TestHubACLDelegatesCommandValidationToHandlers(t *testing.T) {
	assert.True(t, canRunTestCommand(userRoleHub, []string{"SET", "arbitrary", "value"}))
	assert.True(t, canRunTestCommand(userRoleHub, []string{"UNKNOWN"}))
}

func canRunTestCommand(role _UserRole, values []string) bool {
	args := make([][]byte, len(values))
	for index, value := range values {
		args[index] = []byte(value)
	}
	return canRunCommand(role, values[0], args)
}
