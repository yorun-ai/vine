package embedded

import (
	"crypto/subtle"
	"strings"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
)

type _UserRole int

const (
	userRoleNone _UserRole = iota
	userRoleHub
	userRoleLink
	userRolePortal
)

type _ConnContext struct {
	role _UserRole
}

type _ACLRule struct {
	readKeyRules             []string
	readListRules            []string
	subscribeChannelKeyRules []string
}

// aclRuleByRole is the complete resource grant for each non-privileged Redis
// user. In these hard-coded rules, a plain value is an exact key and a trailing
// '*' marks a key prefix. Client-provided keys and patterns are not parsed as
// ACL rules. Commands are checked below, and every resource not listed here is
// denied by default.
var aclRuleByRole = map[_UserRole]_ACLRule{
	userRoleLink: {
		readKeyRules: []string{
			hubredis.RevisionKey,
			"config:*",
			"rpc:*",
		},
		readListRules: []string{
			hubredis.RevisionKey,
			"rpc:*",
		},
		subscribeChannelKeyRules: []string{
			"config:*",
		},
	},
	userRolePortal: {
		readKeyRules: []string{
			hubredis.RevisionKey,
			redised.FormatPortalRulePrefix() + ":*",
			redised.FormatPortalSitePrefix() + ":*",
			redised.FormatPortalCertPrefix() + ":*",
			redised.FormatSchemaActorPrefix() + ":*",
			redised.FormatSchemaServicePrefix() + ":*",
			redised.FormatSchemaResourcePrefix() + ":*",
			"rpc:*",
			"web:*",
		},
		readListRules: []string{
			hubredis.RevisionKey,
			redised.FormatPortalRulePrefix() + ":*",
			redised.FormatPortalSitePrefix() + ":*",
			redised.FormatPortalCertPrefix() + ":*",
			redised.FormatSchemaActorPrefix() + ":*",
			redised.FormatSchemaServicePrefix() + ":*",
			redised.FormatSchemaResourcePrefix() + ":*",
			"rpc:*",
			"web:*",
		},
	},
}

func connContext(conn interface{ Context() any }) *_ConnContext {
	ctx, ok := conn.Context().(*_ConnContext)
	if !ok {
		// A connection has no role until HELLO AUTH succeeds. In particular,
		// anonymous connections must not inherit either read-only client role.
		return &_ConnContext{role: userRoleNone}
	}
	return ctx
}

func (s *Store) authenticate(username string, password string) (_UserRole, bool) {
	switch username {
	case hubredis.HubUsername:
		return authenticatePassword(password, s.hubPassword, userRoleHub)
	case hubredis.LinkUsername:
		return authenticatePassword(password, hubredis.LinkPassword, userRoleLink)
	case hubredis.PortalUsername:
		return authenticatePassword(password, hubredis.PortalPassword, userRolePortal)
	default:
		return userRoleNone, false
	}
}

func authenticatePassword(actual string, expected string, role _UserRole) (_UserRole, bool) {
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return userRoleNone, false
	}
	return role, true
}

// canRunCommand enforces both command and resource permissions. args includes
// the command name at index zero so SCAN can reuse the protocol parser.
func canRunCommand(role _UserRole, command string, args [][]byte) bool {
	if role == userRoleHub {
		return true
	}
	rule, ok := aclRuleByRole[role]
	if !ok {
		return false
	}

	switch strings.ToUpper(command) {
	case "PING":
		return true
	case "GET":
		return len(args) == 2 && rule.canReadKey(string(args[1]))
	case "SCAN":
		option, err := parseScanOption(args)
		return err == nil && rule.canReadPattern(option.match)
	case "SUBSCRIBE":
		return len(args) >= 2 && allArgsAllowed(args[1:], func(value string) bool {
			return rule.canSubscribeChannel(value)
		})
	case "PSUBSCRIBE":
		return len(args) >= 2 && allArgsAllowed(args[1:], func(value string) bool {
			return rule.canReadPattern(value)
		})
	case "UNSUBSCRIBE":
		// Zero channels means unsubscribe from every current subscription. The
		// server only lets this connection create subscriptions inside its ACL.
		return len(args) == 1 || allArgsAllowed(args[1:], func(value string) bool {
			return rule.canSubscribeChannel(value)
		})
	case "PUNSUBSCRIBE":
		return len(args) == 1 || allArgsAllowed(args[1:], func(value string) bool {
			return rule.canReadPattern(value)
		})
	default:
		return false
	}
}

func allArgsAllowed(args [][]byte, allowed func(string) bool) bool {
	for _, arg := range args {
		if !allowed(string(arg)) {
			return false
		}
	}
	return true
}

func (r _ACLRule) canReadKey(key string) bool {
	return matchesAnyKeyRule(key, r.readKeyRules)
}

func (r _ACLRule) canReadPattern(pattern string) bool {
	return matchesAnyKeyRule(pattern, r.readListRules)
}

func (r _ACLRule) canSubscribeChannel(channel string) bool {
	return matchesAnyKeyRule(channel, r.subscribeChannelKeyRules)
}

func matchesAnyKeyRule(value string, rules []string) bool {
	for _, rule := range rules {
		prefix, isPrefix := strings.CutSuffix(rule, "*")
		if isPrefix {
			if strings.HasPrefix(value, prefix) && len(value) > len(prefix) {
				return true
			}
			continue
		}
		if value == rule {
			return true
		}
	}
	return false
}
