package rpcgw

import (
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

func mcpBackend(t *testing.T, watch *watchtest.Client, service string, handler http.HandlerFunc) {
	t.Helper()
	endpoint := "link+inproc://vine/mcp-" + meta.NewId()
	registerTestIngress(t, endpoint, handler)
	watch.SetValue(watched.FormatRpcServiceRegistrationKey(service, "demo.app", "instance-1"), vcode.MustMarshalJsonS(watched.RpcServiceRegistration{
		Endpoint: endpoint, ServiceName: service, AppName: "demo.app", AppInstanceId: "instance-1",
	}))
}

func mcpBackendResult(t *testing.T, w http.ResponseWriter, result any, failure *vrpchttp.ErrorPayload) {
	t.Helper()
	status := ex.OK
	if failure != nil {
		status = ex.Code(failure.Code)
	}
	rpchttp.EncodeStatusCodeToHeader(w.Header(), status)
	rpchttp.EncodeServerToHeader(w.Header(), testServerApp())
	w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
	body, err := vrpchttp.EncodeResponse(result, failure, rpchttp.ContentTypeJson)
	require.NoError(t, err)
	_, _ = w.Write(body)
}

func TestMcpCallAdmissionForwardingAndErrors(t *testing.T) {
	watch := inspectWatch(t)
	actor := &skeldesc.Actor{SkelName: "demo.UserActor", Auth: &skeldesc.ActorAuth{
		Credential: testCredentialDescriptor(), Info: &skeldesc.Data{SkelName: "demo.UserInfo"},
		Service: &skeldesc.Service{SkelName: "demo.AuthService", Methods: []*skeldesc.Method{{Name: "Auth", SkelName: "auth"}}}, MethodName: "Auth",
	}}
	watch.SetValue(watched.FormatDescriptorActorKey(actor.SkelName), vcode.MustMarshalJsonS(actor))
	mcpBackend(t, watch, "demo.AuthService", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var arguments struct {
			Credential map[string]string `json:"credential"`
		}
		require.NoError(t, vrpchttp.DecodeRequest(body, &arguments, r.Header.Get("Content-Type")))
		if arguments.Credential["key"] != "token" {
			mcpBackendResult(t, w, nil, &vrpchttp.ErrorPayload{Code: string(ex.Unauthorized), Message: "invalid token"})
			return
		}
		mcpBackendResult(t, w, map[string]string{"userId": "u1"}, nil)
	})
	var calls atomic.Int32
	var failure atomic.Bool
	var forwardedTrace atomic.Value
	mcpBackend(t, watch, "demo.UserService", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.NoError(t, rpchttp.CheckRequestHeaders(r.Header))
		client, err := rpchttp.DecodeClientFromHeader(r.Header)
		require.NoError(t, err)
		require.Equal(t, "vine.portal", client.Name())
		require.Equal(t, "/demo.UserService/GetV2", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Cookie"))
		require.Empty(t, r.Header.Get("Mcp-Name"))
		rawActor, err := base64.RawURLEncoding.DecodeString(r.Header.Get(rpchttp.HeaderRpcActor))
		require.NoError(t, err)
		var actor struct {
			Type  meta.ActorType `json:"type"`
			Realm string         `json:"realm"`
		}
		require.NoError(t, json.Unmarshal(rawActor, &actor))
		require.Equal(t, meta.ActorTypeAuthenticated, actor.Type)
		require.Equal(t, "demo.UserActor", actor.Realm)
		initiator, err := meta.DecodeInitiatorFromBase64(r.Header.Get(rpchttp.HeaderRpcInitiator))
		require.NoError(t, err)
		require.Equal(t, "vine.mcp", initiator.Name())
		require.Equal(t, "192.0.2.1", initiator.IpAddr())
		trace, err := rpchttp.DecodeTraceFromHeader(r.Header)
		require.NoError(t, err)
		forwardedTrace.Store(trace.Id())
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.Positive(t, time.Until(deadline))
		require.LessOrEqual(t, time.Until(deadline), defaultRpcTimeout)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"params":{"id":9007199254740993}}`, string(body))
		if failure.Load() {
			mcpBackendResult(t, w, nil, &vrpchttp.ErrorPayload{Code: string(ex.OperationFailed), Message: "denied", Reason: "blocked", Detail: "secret"})
			return
		}
		mcpBackendResult(t, w, map[string]string{"name": "Alice"}, nil)
	})
	gateway := newInspectGateway(t, watch, mcpConfig("demo.UserService"))
	call := func(authorization string, args jsontext.Value) *mcp.CallToolResult {
		r := mcpRequest("tools/call", map[string]any{"name": "demo.UserService.GetV2", "arguments": args}, mcpProtocolVersion)
		if authorization != "" {
			r.Header.Set("Authorization", authorization)
		}
		r.Header.Set(rpchttp.HeaderRpcActor, "spoofed")
		r.Header.Set(rpchttp.HeaderRpcInitiator, "spoofed")
		r.Header.Set(rpchttp.HeaderRpcOptions, "timeout=3600000")
		r.Header.Set("Cookie", "private=1")
		response := mcpServe(gateway, r)
		var result mcp.CallToolResult
		mcpResult(t, response, &result)
		if !result.IsError {
			require.Equal(t, forwardedTrace.Load(), response.Header().Get(spec.HeaderPortalTraceId))
			require.Contains(t, response.Body.String(), `"resultType":"complete"`)
		}
		return &result
	}
	args := jsontext.Value(`{"id":9007199254740993}`)
	for _, token := range []string{"", "key bad"} {
		require.True(t, call(token, args).IsError)
	}
	for _, args := range []string{`{}`, `{"id":"wrong"}`, `null`, `[]`} {
		require.True(t, call("key token", jsontext.Value(args)).IsError)
	}
	require.Zero(t, calls.Load())
	result := call("key token", args)
	require.False(t, result.IsError, result.Content)
	require.Equal(t, map[string]any{"name": "Alice"}, result.StructuredContent)
	require.JSONEq(t, `{"name":"Alice"}`, result.Content[0].(*mcp.TextContent).Text)
	failure.Store(true)
	result = call("key token", args)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*mcp.TextContent).Text, "blocked")
	require.NotContains(t, result.Content[0].(*mcp.TextContent).Text, "secret")
	service := inspectService("demo.UserService")
	service.Methods[0].EffectiveRequire = &skeldesc.PermissionRequire{Expression: &skeldesc.PermissionExpression{Mode: skeldesc.PermissionRequireModeCode, Code: "demo.User:read"}}
	watch.Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorServiceKey(service.SkelName), Value: vcode.MustMarshalJsonS(service)})
	require.Eventually(t, func() bool {
		snapshot := gateway.desc.Snapshot()
		return !snapshot.Syncing && snapshot.Services[service.SkelName].Methods[0].EffectiveRequire != nil
	}, time.Second, time.Millisecond)
	count := calls.Load()
	result = call("key token", args)
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*mcp.TextContent).Text, "permission service")
	require.Equal(t, count, calls.Load())
}

func TestMcpCallCancellationReachesBackend(t *testing.T) {
	watch := newTestDescriptorWatch(t)
	entered, cancelled := make(chan struct{}), make(chan struct{})
	mcpBackend(t, watch, "demo.UserService", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	})
	gateway := newInspectGateway(t, watch, mcpConfig("demo.UserService"))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	r := mcpRequest("tools/call", map[string]any{"name": "demo.UserService.Get"}, mcpProtocolVersion).WithContext(ctx)
	done := make(chan struct{})
	go func() { mcpServe(gateway, r); close(done) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not start")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("backend was not cancelled")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("MCP request did not finish")
	}
}

func TestMcpStructuredResultsPreserveTypesAndValidate(t *testing.T) {
	for _, test := range []struct {
		name  string
		kind  *skeldesc.Type
		value jsontext.Value
	}{
		{"integer", &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt}, jsontext.Value(`9007199254740993`)},
		{"string", &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString}, jsontext.Value(`"hello"`)},
		{"boolean", &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBoolean}, jsontext.Value(`false`)},
		{"array", &skeldesc.Type{Kind: skeldesc.TypeKindList, Element: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString}}, jsontext.Value(`["a"]`)},
		{"void", nil, jsontext.Value(`null`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			watch := newTestDescriptorWatch(t)
			service := inspectService("demo.UserService")
			service.Methods[0].Arguments = nil
			service.Methods[0].EffectiveAuthMode = skeldesc.AuthModeOptional
			service.Methods[0].ResultType = test.kind
			watch.SetValue(watched.FormatDescriptorServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
			var invalid atomic.Bool
			mcpBackend(t, watch, service.SkelName, func(w http.ResponseWriter, r *http.Request) {
				if invalid.Load() {
					mcpBackendResult(t, w, map[string]any{}, nil)
					return
				}
				mcpBackendResult(t, w, test.value, nil)
			})
			gateway := newInspectGateway(t, watch, mcpConfig(service.SkelName))
			response := mcpServe(gateway, mcpRequest("tools/call", map[string]any{"name": "demo.UserService.GetV2"}, "2026-07-28"))
			var result mcp.CallToolResult
			mcpResult(t, response, &result)
			require.False(t, result.IsError, result.Content)
			require.Equal(t, string(test.value), result.Content[0].(*mcp.TextContent).Text)
			var wire struct {
				Result struct {
					StructuredContent jsontext.Value `json:"structuredContent"`
				} `json:"result"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &wire))
			require.Equal(t, string(test.value), string(wire.Result.StructuredContent))
			var listing mcp.ListToolsResult
			mcpResult(t, mcpServe(gateway, mcpRequest("tools/list", nil, mcpProtocolVersion)), &listing)
			require.Len(t, listing.Tools, 1)
			output, err := resolveMcpSchema(listing.Tools[0].OutputSchema)
			require.NoError(t, err)
			require.NoError(t, validateMcpValue(output, wire.Result.StructuredContent))
			invalid.Store(true)
			mcpResult(t, mcpServe(gateway, mcpRequest("tools/call", map[string]any{"name": "demo.UserService.GetV2"}, "2026-07-28")), &result)
			require.True(t, result.IsError)
		})
	}
}
