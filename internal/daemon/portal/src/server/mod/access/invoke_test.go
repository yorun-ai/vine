package access

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/mtls"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
)

func authForwarder(t *testing.T, endpoint string) *Auther {
	t.Helper()
	const service = "demo.AuthService"
	m := &epmgr.Manager{
		Context: t.Context(),
		Watch: watchtest.New(t, map[string]string{
			watched.FormatRpcServiceRegistrationKey(service, "demo", uuid.MustParse("00000000-0000-0000-0000-00000000000a")): vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
				Endpoint:      endpoint,
				ServiceName:   service,
				AppName:       "demo",
				AppInstanceId: uuid.MustParse("00000000-0000-0000-0000-00000000000a"),
			}),
		}),
	}
	m.DIInit()
	w := m.WatchRpc(service)
	t.Cleanup(w.Release)
	return &Auther{endpointManager: m, identity: mtls.DisabledIdentity()}
}

func TestAuthAndPermissionForwardingReportConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	endpoint := "http://" + listener.Addr().String()
	require.NoError(t, listener.Close())
	operation := authForwarder(t, endpoint)
	request := httptest.NewRequest(http.MethodPost, "http://portal.local/invoke", nil)
	_, code, message, ok := operation.forwardInvokeRequest(request, "demo.AuthService", "auth")
	require.False(t, ok)
	require.Equal(t, ex.ServiceUnavailable, code)
	require.Contains(t, message, "forward failed")
	_, _, message, ok = operation.forwardInvokeRequest(request, "demo.AuthService", "permission")
	require.False(t, ok)
	require.Contains(t, message, "is unavailable")
}

func TestAccessBusinessErrorsDoNotIsolateEndpoint(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("denied"))
	}))
	server.Config.Protocols = new(http.Protocols)
	server.Config.Protocols.SetUnencryptedHTTP2(true)
	server.Start()
	t.Cleanup(server.Close)
	operation := authForwarder(t, server.URL)
	for range 5 {
		request := httptest.NewRequest(http.MethodPost, "http://portal.local/invoke", nil)
		response, _, _, ok := operation.forwardInvokeRequest(request, "demo.AuthService", "auth")
		require.True(t, ok)
		require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		_, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
	}
	require.Equal(t, int32(5), calls.Load())
}

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
		result, _, _, _, ok := readInvokeResponse[benchmarkInvokeResult](response, "benchmark", "bad response", "failed")
		if !ok || !result.Allowed || len(result.Roles) != 2 {
			b.Fatal("readInvokeResponse() returned an unexpected result")
		}
		_ = response.Body.Close()
	}
}
