package rpcgw

import (
	"net/http"
	"strings"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

func prepareForwardRequest(request *http.Request) (string, *http.Request) {
	acceptEncoding := request.Header.Get(rpchttp.HeaderAcceptEncoding)
	next := request.Clone(request.Context())
	next.Trailer = nil
	next.TransferEncoding = nil
	next.Close = false
	next.Host = ""
	// Admission has already reconstructed actor and initiator metadata. Only
	// the vRPC protocol contract crosses the backend boundary.
	next.Header = make(http.Header)
	for _, name := range []string{
		rpchttp.HeaderAccept, rpchttp.HeaderContentType, rpchttp.HeaderRpcClient,
		rpchttp.HeaderRpcTrace, rpchttp.HeaderRpcInitiator, rpchttp.HeaderRpcActor,
	} {
		if values := request.Header.Values(name); len(values) > 0 {
			next.Header[http.CanonicalHeaderKey(name)] = append([]string(nil), values...)
		}
	}
	// An explicit empty User-Agent prevents Go transports from synthesizing one.
	// This is a transport directive, not client metadata sent to the backend.
	next.Header["User-Agent"] = []string{""}
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

// Requests are forwarded without decoding Content-Encoding. Never strip an
// unsupported encoding and reinterpret its bytes as an ordinary RPC body.
func supportedRequestContentEncoding(header http.Header) bool {
	values := header.Values(rpchttp.HeaderContentEncoding)
	return len(values) == 0 || (len(values) == 1 && strings.EqualFold(strings.TrimSpace(values[0]), "identity"))
}
