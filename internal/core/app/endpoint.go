package app

import "uuid"

const (
	PathConsole   = "/console"
	PathRpcInvoke = "/rpc/invoke"
	PathWebAccess = "/web/access"
	PathEvent     = "/event"
	PathTask      = "/task"
)

func InprocHostPath(instanceId uuid.UUID) string {
	return "app/" + instanceId.String()
}
