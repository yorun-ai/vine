package redised

import "fmt"

const (
	rpcServiceRegistrationPrefixFormat = "rpc:%s:endpoint"
	rpcServiceRegistrationKeyFormat    = rpcServiceRegistrationPrefixFormat + ":%s:%s"
)

type RpcServiceRegistration struct {
	Endpoint      string `json:"endpoint"`
	ServiceName   string `json:"serviceName"`
	AppName       string `json:"appName"`
	AppVersion    string `json:"appVersion"`
	AppInstanceId string `json:"appInstanceId"`
}

func FormatRpcServiceRegistrationKey(serviceName string, appName string, instanceId string) string {
	return fmt.Sprintf(rpcServiceRegistrationKeyFormat, serviceName, appName, instanceId)
}

func FormatRpcServiceRegistrationPrefix(serviceName string) string {
	return fmt.Sprintf(rpcServiceRegistrationPrefixFormat, serviceName)
}

const (
	appStatusPrefixFormat = "app:%s:status"
	appStatusKeyFormat    = appStatusPrefixFormat + ":%s"
	appStatusPattern      = "app:*:status:*"
)

func FormatAppStatusKey(appName string, instanceId string) string {
	return fmt.Sprintf(appStatusKeyFormat, appName, instanceId)
}

func FormatAppStatusPattern() string {
	return appStatusPattern
}
