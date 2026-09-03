package access

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
)

type benchmarkInvokeResult struct {
	Allowed bool     `json:"allowed"`
	ActorId string   `json:"actorId"`
	Roles   []string `json:"roles"`
}

func BenchmarkReadInvokeResponse(b *testing.B) {
	body := []byte(`{"result":{"allowed":true,"actorId":"123e4567-e89b-12d3-a456-426614174000","roles":["admin","operator"]},"error":null}`)
	header := make(http.Header)
	rpchttp.EncodeStatusCodeToHeader(header, ex.OK)

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		response := &http.Response{
			StatusCode: http.StatusOK,
			Header:     header,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}
		result, _, _, ok := readInvokeResponse[benchmarkInvokeResult](response, "benchmark", "bad response", "failed")
		if !ok || !result.Allowed || len(result.Roles) != 2 {
			b.Fatal("readInvokeResponse() returned an unexpected result")
		}
		_ = response.Body.Close()
	}
}
