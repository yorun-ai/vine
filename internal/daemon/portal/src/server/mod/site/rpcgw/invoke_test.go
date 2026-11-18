package rpcgw

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
)

func endpointValue(instance string, endpoint string) string {
	return vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
		Endpoint:      endpoint,
		ServiceName:   "demo.UserService",
		AppName:       "demo.app",
		AppInstanceId: uuid.MustParse(instance),
	})
}

func newH2Backend(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.Config.Protocols = new(http.Protocols)
	server.Config.Protocols.SetHTTP1(true)
	server.Config.Protocols.SetUnencryptedHTTP2(true)
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func invokeForIsolationTest(t *testing.T, gateway *RpcGateway) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/invoke/demo.UserService/Get", strings.NewReader(`{}`))
	setTestAuthHeaders(request)
	recorder := httptest.NewRecorder()
	gateway.Serve(testContext(recorder, request))
	return recorder
}

func TestRpcGatewaySkipsRefusedEndpointWithoutRetryingCurrentRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadEndpoint := "http://" + listener.Addr().String()
	require.NoError(t, listener.Close())
	var calls atomic.Int32
	backend := newH2Backend(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	target := newTestRpcGateway(t, map[string]string{
		watched.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000066")): endpointValue("00000000-0000-0000-0000-000000000066", deadEndpoint),
		watched.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", uuid.MustParse("00000000-0000-0000-0000-00000000006a")): endpointValue("00000000-0000-0000-0000-00000000006a", backend.URL),
	})
	t.Cleanup(target.Stop)
	statuses := map[int]int{}
	for range 3 {
		statuses[invokeForIsolationTest(t, target).Code]++
	}
	require.Equal(t, map[int]int{http.StatusServiceUnavailable: 1, http.StatusAccepted: 2}, statuses)
	require.Equal(t, int32(2), calls.Load(), "a failed request must not be replayed against the healthy endpoint")
}

func TestRpcGatewayCountsTruncatedResponsesWithoutReexecutingRequests(t *testing.T) {
	var executions atomic.Int32
	backend := newH2Backend(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executions.Add(1)
		// The application executed the request, but the response was truncated.
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("{"))
	}))
	target := newTestRpcGateway(t, map[string]string{
		watched.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", uuid.MustParse("00000000-0000-0000-0000-000000000065")): endpointValue("00000000-0000-0000-0000-000000000065", backend.URL),
	})
	t.Cleanup(target.Stop)
	for range 3 {
		response := invokeForIsolationTest(t, target)
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		require.Contains(t, response.Body.String(), "response body cannot be read")
	}
	response := invokeForIsolationTest(t, target)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "endpoint is unavailable")
	require.Equal(t, int32(3), executions.Load())
}
