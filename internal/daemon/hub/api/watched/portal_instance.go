package watched

import (
	"fmt"
	"time"
	"uuid"
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
	InstanceId uuid.UUID `json:"instanceId"`
	Version    string    `json:"version"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func FormatPortalInstanceKey(instanceId uuid.UUID) string {
	return fmt.Sprintf(portalInstanceKeyFormat, instanceId)
}

func FormatPortalInstancePattern() string {
	return portalInstancePattern
}
