package spec

type RpcHandler interface {
	ServeRpc(rpcRequest Request) Response
}

type RpcHandlerFunc func(rpcRequest Request) Response

func (f RpcHandlerFunc) ServeRpc(rpcRequest Request) Response {
	return f(rpcRequest)
}
