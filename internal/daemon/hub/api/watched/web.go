package watched

import (
	"fmt"
	"uuid"

	"go.yorun.ai/vine/internal/daemon"
)

const (
	webRegistrationPrefixFormat = "web:%s:endpoint"
	webRegistrationKeyFormat    = webRegistrationPrefixFormat + ":%s:%s"
)

type WebRegistration struct {
	Endpoint       string          `json:"endpoint"`
	ServerIdentity daemon.Identity `json:"serverIdentity,omitempty"`
	WebSkelName    string          `json:"webSkelName"`
	AppName        string          `json:"appName"`
	AppVersion     string          `json:"appVersion"`
	AppInstanceId  uuid.UUID       `json:"appInstanceId"`
}

func FormatWebRegistrationKey(webSkelName string, appName string, instanceId uuid.UUID) string {
	return fmt.Sprintf(webRegistrationKeyFormat, webSkelName, appName, instanceId)
}

func FormatWebRegistrationPrefix(webSkelName string) string {
	return fmt.Sprintf(webRegistrationPrefixFormat, webSkelName)
}
