package http

import (
	"fmt"
	"io"
	"net/http"
)

const (
	// MaxRequestBodyBytes is the maximum encoded Rpc request body size.
	MaxRequestBodyBytes int64 = 32 << 20
	// MaxResponseBodyBytes is the maximum encoded Rpc response body size.
	MaxResponseBodyBytes int64 = 128 << 20
)

// ReadRequestBody reads the Rpc request body up to the fixed transport limit.
func ReadRequestBody(request *http.Request) ([]byte, error) {
	return readBody(request.Body, request.ContentLength, MaxRequestBodyBytes, "request")
}

// ReadResponseBody reads the Rpc response body up to the fixed transport limit.
func ReadResponseBody(response *http.Response) ([]byte, error) {
	return readBody(response.Body, response.ContentLength, MaxResponseBodyBytes, "response")
}

func readBody(body io.Reader, contentLength int64, limit int64, scope string) ([]byte, error) {
	if contentLength > limit {
		return nil, fmt.Errorf("rpc %s body exceeds %d byte limit", scope, limit)
	}
	if body == nil {
		return []byte{}, nil
	}

	content, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("rpc %s body exceeds %d byte limit", scope, limit)
	}
	return content, nil
}
