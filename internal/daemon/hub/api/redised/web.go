package redised

import (
	"fmt"

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
	AppInstanceId  string          `json:"appInstanceId"`
}

func FormatWebRegistrationKey(webSkelName string, appName string, instanceId string) string {
	return fmt.Sprintf(webRegistrationKeyFormat, webSkelName, appName, instanceId)
}

func FormatWebRegistrationPrefix(webSkelName string) string {
	return fmt.Sprintf(webRegistrationPrefixFormat, webSkelName)
}
