package rpcgw

import (
	"net/http"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

func prepareForwardRequest(request *http.Request) (string, *http.Request) {
	acceptEncoding := request.Header.Get(rpchttp.HeaderAcceptEncoding)
	next := request.Clone(request.Context())
	next.Header = request.Header.Clone()
	next.Header.Del(rpchttp.HeaderAcceptEncoding)
	next.Header.Del(rpchttp.HeaderRpcOptions)
	rpchttp.EncodeRequestOptionsToHeader(next.Header, next.Context())
	encodeForwardTrace(next.Header)
	return acceptEncoding, next
}

func encodeForwardTrace(header http.Header) {
	trace, err := rpchttp.DecodeTraceFromHeader(header)
	if err != nil {
		return
	}
	rpchttp.EncodeTraceToHeader(header, trace.NewChildTrace())
}
