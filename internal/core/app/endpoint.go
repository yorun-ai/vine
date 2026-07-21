package app

const (
	PathConsole   = "/console"
	PathRpcInvoke = "/rpc/invoke"
	PathWebAccess = "/web/access"
	PathEvent     = "/event"
	PathTask      = "/task"
)

func InprocHostPath(instanceId string) string {
	return "app/" + instanceId
}
