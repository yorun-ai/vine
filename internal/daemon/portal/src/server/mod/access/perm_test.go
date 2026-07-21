package access

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/util/vcode"
)

func TestCheckActorPermissionsRejectsMissingCodeResult(t *testing.T) {
	serverApp := meta.MustNewApp("perm.test", "0.0.0", "123e4567-e89b-12d3-a456-426614174000")
	permissionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &spec.ResponseImpl{
			ServerValue: serverApp,
			ErrorValue:  ex.NewOK(),
			ResultValue: map[string]bool{
				"app.User:read": true,
			},
		}
		if err := rpchttp.WriteResponse(w, r, response); err != nil {
			t.Fatalf("WriteResponse() error = %v", err)
		}
	}))
	t.Cleanup(permissionServer.Close)

	manager := newAccessTestEndpointManager("app.UserActorPermissionService", permissionServer.URL)
	watcher := manager.WatchRpc("app.UserActorPermissionService")
	t.Cleanup(watcher.Release)

	initiator, err := meta.NewInitiator("demo.client", "0.0.0", "123e4567-e89b-12d3-a456-426614174001", "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("NewInitiator() error = %v", err)
	}
	request := rpchttp.BuildInvokeRequest(rpchttp.InvokeRequest{
		Context:         context.Background(),
		Endpoint:        "https://demo.local/rpc/invoke",
		ServiceSkelName: "app.UserService",
		MethodSkelName:  "getUser",
		Trace:           meta.InitialTrace(),
		Client:          initiator,
		Initiator:       initiator,
	})
	recorder := httptest.NewRecorder()

	operation := &RpcOperation{
		Auther: Auther{
			Request:         request,
			Response:        recorder,
			Trace:           meta.InitialTrace(),
			Initiator:       initiator,
			endpointManager: manager,
			actorSchema: &skel.ActorSchema{
				PermEnabled: true,
				PermService: &skel.ServiceSchema{SkelName: "app.UserActorPermissionService"},
				PermMethod:  &skel.MethodSchema{SkelName: "checkCodes"},
			},
		},
		Server: serverApp,
	}

	ok := operation.checkActorPermissions(&skel.PermExpr{
		Mode: skel.PermRequireModeAll,
		Children: []*skel.PermExpr{
			{Mode: skel.PermRequireModeCode, Code: "app.User:read"},
			{Mode: skel.PermRequireModeCode, Code: "app.User:manage"},
		},
	})
	if ok {
		t.Fatalf("checkActorPermissions() ok = true, want false")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected http status: %d", recorder.Code)
	}
	statusCode, err := rpchttp.DecodeStatusCodeFromHeader(recorder.Header())
	if err != nil {
		t.Fatalf("DecodeStatusCodeFromHeader() error = %v", err)
	}
	if statusCode != ex.ServiceUnavailable {
		t.Fatalf("unexpected rpc status: %s", statusCode)
	}
}

func TestExtractCheckParamsSupportsCborRequestBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/rpc/invoke/app.UserService/update", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeCbor)
	operation := &RpcOperation{
		Auther: Auther{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
		Server: meta.MustNewApp("perm.test", "0.0.0", "123e4567-e89b-12d3-a456-426614174021"),
		requestBody: vcode.MustMarshalCbor(map[string]any{
			"params": map[string]any{
				"update": map[string]any{
					"userId": 42,
				},
				"items": []any{
					map[string]any{"id": "first"},
					map[string]any{"id": "second"},
				},
			},
		}),
	}

	params, ok := operation.extractCheckParams(&skel.PermCheckInvocation{
		ResourceSkelName: "app.User",
		ActionName:       "update",
		Arguments: []*skel.PermCheckArgument{
			{Name: "userId", JsonPath: "update.userId"},
			{Name: "itemIds", JsonPath: "items[*].id"},
		},
	})
	if !ok {
		t.Fatalf("extractCheckParams() ok = false, want true")
	}
	if params["code"] != skel.PermissionCode("app.User:update") {
		t.Fatalf("unexpected code param: %#v", params["code"])
	}
	if params["userId"] != uint64(42) {
		t.Fatalf("unexpected userId param: %#v", params["userId"])
	}
	itemIds := params["itemIds"].([]any)
	if len(itemIds) != 2 || itemIds[0] != "first" || itemIds[1] != "second" {
		t.Fatalf("unexpected itemIds param: %#v", params["itemIds"])
	}
}

func TestExtractCheckParamsSupportsJsonWildcardPath(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/rpc/invoke/app.UserService/update", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	operation := &RpcOperation{
		Auther: Auther{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
		Server: meta.MustNewApp("perm.test", "0.0.0", "123e4567-e89b-12d3-a456-426614174020"),
		requestBody: vcode.MustMarshalJson(map[string]any{
			"params": map[string]any{
				"items": []any{
					map[string]any{"id": "first"},
					map[string]any{"id": "second"},
				},
			},
		}),
	}

	params, ok := operation.extractCheckParams(&skel.PermCheckInvocation{
		ResourceSkelName: "app.User",
		ActionName:       "update",
		Arguments: []*skel.PermCheckArgument{
			{Name: "itemIds", JsonPath: "items[*].id"},
		},
	})
	if !ok {
		t.Fatalf("extractCheckParams() ok = false, want true")
	}
	itemIds := params["itemIds"].([]any)
	if len(itemIds) != 2 || itemIds[0] != "first" || itemIds[1] != "second" {
		t.Fatalf("unexpected itemIds param: %#v", params["itemIds"])
	}
}

func TestExtractCheckParamsRejectsTrailingWildcardPath(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/rpc/invoke/app.UserService/update", nil)
	request.Header.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	operation := &RpcOperation{
		Auther: Auther{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
		Server: meta.MustNewApp("perm.test", "0.0.0", "123e4567-e89b-12d3-a456-426614174022"),
		requestBody: vcode.MustMarshalJson(map[string]any{
			"params": map[string]any{
				"items": []any{
					map[string]any{"id": "first"},
				},
			},
		}),
	}

	_, ok := operation.extractCheckParams(&skel.PermCheckInvocation{
		ResourceSkelName: "app.User",
		ActionName:       "update",
		Arguments: []*skel.PermCheckArgument{
			{Name: "items", JsonPath: "items[*]"},
		},
	})
	if ok {
		t.Fatalf("extractCheckParams() ok = true, want false")
	}
}

func newAccessTestEndpointManager(serviceName string, endpoint string) *epmgr.Manager {
	manager := &epmgr.Manager{
		Context: context.Background(),
		Redis: hubredis.NewTestClient(map[string]string{
			redised.FormatRpcServiceRegistrationKey(serviceName, "perm.test", "instance-1"): vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
				Endpoint:      endpoint,
				ServiceName:   serviceName,
				AppName:       "perm.test",
				AppInstanceId: "instance-1",
			}),
		}),
	}
	manager.DIInit()
	return manager
}
