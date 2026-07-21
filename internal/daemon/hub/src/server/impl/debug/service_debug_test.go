package debug

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

type _ServiceDebugRegistryRepo struct {
	core.RegistryRepo
	status       *core.AppStatus
	registration *core.RpcServiceRegistration
}

func (r *_ServiceDebugRegistryRepo) GetAppStatus(appName string, instanceId string) (*core.AppStatus, bool) {
	if r.status.Name == appName && r.status.InstanceId == instanceId {
		return r.status, true
	}
	return nil, false
}

func (r *_ServiceDebugRegistryRepo) GetRpcServiceRegistration(serviceName string, appName string, instanceId string) (*core.RpcServiceRegistration, bool) {
	if r.registration.ServiceName == serviceName && r.registration.AppName == appName && r.registration.AppInstanceId == instanceId {
		return r.registration, true
	}
	return nil, false
}

func TestInvokeServicePropagatesTimeoutAsRpcOptions(t *testing.T) {
	ingressEndpoint := "link+inproc://vine/hub-debug-timeout-test"
	ingressinproc.Register(ingressEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Debug-Rpc-Options", r.Header.Get(rpchttp.HeaderRpcOptions))
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(func() { ingressinproc.Unregister(ingressEndpoint) })

	appName := "demo.app"
	appInstanceId := "instance-1"
	target := &ServiceDebugServiceServerImpl{
		RegistryRepo: &_ServiceDebugRegistryRepo{
			status: &core.AppStatus{
				Name:       appName,
				InstanceId: appInstanceId,
			},
			registration: &core.RpcServiceRegistration{
				Endpoint:      ingressEndpoint,
				ServiceName:   "demo.UserService",
				AppName:       appName,
				AppInstanceId: appInstanceId,
			},
		},
	}

	response := target.InvokeService(skeled.ServiceDebugInvokeRequest{
		AppName:         &appName,
		AppInstanceId:   &appInstanceId,
		ServiceSkelName: "demo.UserService",
		MethodSkelName:  "Get",
		ParamsJson:      skel.JSON(`{}`),
		TimeoutSeconds:  5,
	})

	if response.HttpStatus != http.StatusAccepted {
		t.Fatalf("http status = %d, want %d", response.HttpStatus, http.StatusAccepted)
	}
	header := http.Header{}
	if err := json.Unmarshal([]byte(response.HeadersJson), &header); err != nil {
		t.Fatalf("Unmarshal headers error = %v", err)
	}
	options := mustDecodeDebugRpcOptions(t, header.Get("X-Debug-Rpc-Options"))
	if options.Timeout <= 0 || options.Timeout > 5*time.Second {
		t.Fatalf("timeout = %s, want within 5s", options.Timeout)
	}
}

func mustDecodeDebugRpcOptions(t *testing.T, value string) *rpchttp.Options {
	t.Helper()

	header := http.Header{}
	header.Set(rpchttp.HeaderRpcOptions, value)
	options, err := rpchttp.DecodeOptionsFromHeader(header)
	if err != nil {
		t.Fatalf("DecodeOptionsFromHeader() error = %v", err)
	}
	return options
}
