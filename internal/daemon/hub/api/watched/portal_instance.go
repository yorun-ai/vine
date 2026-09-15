package watched

import (
	"fmt"
	"time"
)

// Portal instances

const (
	portalInstancePrefix    = "portal:instance"
	portalInstanceKeyFormat = portalInstancePrefix + ":%s"
	portalInstancePattern   = portalInstancePrefix + ":*"
)

// PortalInstance is a Portal daemon instance registered with Hub. Hub grants the
// watch ACL roles per prefix, and Portal only reads portal:rule, portal:site and
// portal:cert, so this Hub-owned record stays invisible to Portal clients.
type PortalInstance struct {
	InstanceId string    `json:"instanceId"`
	Version    string    `json:"version"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func FormatPortalInstanceKey(instanceId string) string {
	return fmt.Sprintf(portalInstanceKeyFormat, instanceId)
}

func FormatPortalInstancePattern() string {
	return portalInstancePattern
}
