package rpcgw

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
)

func mcpConfig(names ...string) watched.PortalSite {
	config := inspectConfig(names...)
	config.RpcgwConfig.InspectEnabled = false
	config.RpcgwConfig.McpEnabled = true
	return config
}

func mcpRequest(method string, params any, version string) *http.Request {
	if version >= mcpProtocolVersion {
		if params == nil {
			params = map[string]any{}
		}
		params.(map[string]any)["_meta"] = map[string]any{
			mcp.MetaKeyProtocolVersion:    version,
			mcp.MetaKeyClientCapabilities: map[string]any{},
			mcp.MetaKeyClientInfo:         map[string]string{"name": "Test MCP Client", "version": "dev"},
		}
	}
	body := vcode.MustMarshalJsonS(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	r := httptest.NewRequest(http.MethodPost, "http://api.example.com/mcp", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("MCP-Protocol-Version", version)
	r.Header.Set("Mcp-Method", method)
	if p, ok := params.(map[string]any); ok {
		if name, ok := p["name"].(string); ok {
			r.Header.Set("Mcp-Name", name)
		}
	}
	return r
}

func mcpServe(gateway *RpcGateway, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	gateway.Serve(testContextWithEntryOrigin(response, request, "api.example.com"))
	return response
}

func mcpResult(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Empty(t, response.Header().Get("Vrpc-Status"))
	var envelope struct {
		Result jsontext.Value `json:"result"`
		Error  jsontext.Value `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Empty(t, envelope.Error, response.Body.String())
	require.NoError(t, json.Unmarshal(envelope.Result, target))
}

func TestMcpSDKClientDiscoveryAndPagination(t *testing.T) {
	watch := inspectWatch(t)
	service := inspectService("demo.UserService")
	for i := range mcpPageSize {
		service.Methods = append(service.Methods, &skeldesc.Method{Name: fmt.Sprintf("M%03d", i), SkelName: fmt.Sprintf("M%03d", i), EffectiveAuthMode: skeldesc.AuthModeOptional})
	}
	watch.SetValue(watched.FormatDescriptorServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
	mcpBackend(t, watch, service.SkelName, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/demo.UserService/M000", r.URL.Path)
		mcpBackendResult(t, w, nil, nil)
	})
	gateway := newInspectGateway(t, watch, mcpConfig(service.SkelName))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.Serve(testContextWithEntryOrigin(w, r, "api.example.com"))
	}))
	t.Cleanup(server.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "vine-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp"}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	first, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, first.Tools, mcpPageSize)
	require.NotEmpty(t, first.NextCursor)
	require.Equal(t, "demo.UserService.GetV2", first.Tools[0].Name)
	input, err := resolveMcpSchema(first.Tools[0].InputSchema)
	require.NoError(t, err)
	require.NoError(t, validateMcpValue(input, []byte(`{"id":1}`)))
	output, err := resolveMcpSchema(first.Tools[0].OutputSchema)
	require.NoError(t, err)
	require.NoError(t, validateMcpValue(output, []byte(`{"name":"Alice"}`)))
	second, err := session.ListTools(t.Context(), &mcp.ListToolsParams{Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Tools, 1)
	require.Empty(t, second.NextCursor)
	_, err = session.ListTools(t.Context(), &mcp.ListToolsParams{Cursor: "invalid"})
	require.Error(t, err)
	called, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "demo.UserService.M000"})
	require.NoError(t, err)
	require.False(t, called.IsError, called.Content)
	require.Nil(t, called.StructuredContent)
}

func TestMcpSwitchScopeAndOrigin(t *testing.T) {
	watch := inspectWatch(t)
	for _, name := range []string{"demo.Hidden", "demo.Internal", "demo.OtherActor"} {
		service := inspectService(name)
		if name == "demo.Internal" {
			service.Api = false
		}
		if name == "demo.OtherActor" {
			service.Audiences[0].SkelName = "demo.OtherActor"
		}
		watch.SetValue(watched.FormatDescriptorServiceKey(name), vcode.MustMarshalJsonS(service))
	}
	config := mcpConfig("demo.UserService", "demo.Internal", "demo.OtherActor")
	gateway := newInspectGateway(t, watch, config)
	var result mcp.ListToolsResult
	mcpResult(t, mcpServe(gateway, mcpRequest("tools/list", nil, mcpProtocolVersion)), &result)
	require.Len(t, result.Tools, 1)
	require.Equal(t, http.StatusNotFound, inspectRequest(gateway, "GET", "/inspect/_list").Code)
	for _, origin := range []string{"https://evil.test", "null", "http://user@api.example.com", "https://api.example.com:443/path"} {
		request := mcpRequest("tools/list", nil, mcpProtocolVersion)
		request.Header.Set("Origin", origin)
		require.Equal(t, http.StatusForbidden, mcpServe(gateway, request).Code, origin)
		request.Method = http.MethodOptions
		require.Equal(t, http.StatusForbidden, mcpServe(gateway, request).Code, origin)
	}
	request := mcpRequest("tools/list", nil, mcpProtocolVersion)
	request.Method = http.MethodOptions
	request.Header.Set("Origin", "https://console.example.com")
	response := mcpServe(gateway, request)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "POST, OPTIONS", response.Header().Get(spec.HeaderAccessControlAllowMethods))
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request := mcpRequest("tools/list", nil, mcpProtocolVersion)
		request.Method = method
		require.Equal(t, http.StatusMethodNotAllowed, mcpServe(gateway, request).Code)
	}
	config.ActorSkelName = "demo.MissingActor"
	require.True(t, gateway.Update(config))
	var empty mcp.ListToolsResult
	mcpResult(t, mcpServe(gateway, mcpRequest("tools/list", nil, "2026-07-28")), &empty)
	require.Empty(t, empty.Tools)
	config.RpcgwConfig.McpEnabled = false
	gateway.Update(config)
	require.Equal(t, http.StatusNotFound, mcpServe(gateway, mcpRequest("tools/list", nil, mcpProtocolVersion)).Code)
}

func TestMcpOnlyModernProtocol(t *testing.T) {
	gateway := newInspectGateway(t, inspectWatch(t), mcpConfig("demo.UserService"))
	var discovery mcp.DiscoverResult
	mcpResult(t, mcpServe(gateway, mcpRequest("server/discover", nil, mcpProtocolVersion)), &discovery)
	require.Equal(t, []string{mcpProtocolVersion}, discovery.SupportedVersions)
	require.NotNil(t, discovery.Capabilities.Tools)
	require.False(t, discovery.Capabilities.Tools.ListChanged)

	for _, version := range []string{"", "2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25", "2099-01-01"} {
		t.Run("reject_"+version, func(t *testing.T) {
			request := mcpRequest("tools/list", nil, version)
			if version == "" {
				request = mcpRequest("initialize", map[string]any{
					"protocolVersion": "2025-11-25", "capabilities": map[string]any{},
					"clientInfo": map[string]string{"name": "legacy", "version": "1"},
				}, "")
			}
			response := mcpServe(gateway, request)
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			var envelope struct {
				Error struct {
					Code int                                `json:"code"`
					Data mcp.UnsupportedProtocolVersionData `json:"data"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			code := mcp.CodeUnsupportedProtocolVersion
			if version == "" {
				code = mcp.CodeHeaderMismatch
			}
			require.EqualValues(t, code, envelope.Error.Code)
			require.Equal(t, []string{mcpProtocolVersion}, envelope.Error.Data.Supported)
			require.Equal(t, version, envelope.Error.Data.Requested)
		})
	}

	// A modern header must not allow a legacy handshake to bypass metadata checks.
	request := mcpRequest("initialize", map[string]any{"protocolVersion": "2025-11-25"}, "")
	request.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
	require.Equal(t, http.StatusBadRequest, mcpServe(gateway, request).Code)
	request = mcpRequest("initialize", map[string]any{"protocolVersion": mcpProtocolVersion}, mcpProtocolVersion)
	response := mcpServe(gateway, request)
	require.Contains(t, response.Body.String(), `"error"`)
}

func TestMcpProtocolErrors(t *testing.T) {
	gateway := newInspectGateway(t, inspectWatch(t), mcpConfig("demo.UserService"))
	for _, name := range []string{"demo.Hidden.GetV2", "demo.UserService.Get"} {
		response := mcpServe(gateway, mcpRequest("tools/call", map[string]any{"name": name}, mcpProtocolVersion))
		require.Contains(t, response.Body.String(), `"code":-32602`)
	}
	r := mcpRequest("tools/list", nil, mcpProtocolVersion)
	r.Header.Set("Mcp-Method", "tools/call")
	require.Equal(t, http.StatusBadRequest, mcpServe(gateway, r).Code)
	for _, body := range []string{`{`, `[]`, `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`} {
		r = httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		r.Header.Set("MCP-Protocol-Version", mcpProtocolVersion)
		r.Header.Set("Mcp-Method", "tools/list")
		response := mcpServe(gateway, r)
		require.True(t, response.Code >= 400 || strings.Contains(response.Body.String(), `"error"`), response.Body.String())
	}
}

func TestMcpSyncingAndDescriptorRecovery(t *testing.T) {
	client := inspectWatch(t)
	watch := &inspectBlockingWatch{ClientOps: client, entered: make(chan struct{}), release: make(chan struct{})}
	gateway := newInspectGateway(t, watch, mcpConfig("demo.UserService"))
	service := inspectService("demo.UserService")
	service.Methods[0].ResultType.SkelName = "demo.Missing"
	client.Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorServiceKey(service.SkelName), Value: vcode.MustMarshalJsonS(service)})
	select {
	case <-watch.entered:
	case <-time.After(time.Second):
		t.Fatal("type refresh did not start")
	}
	response := mcpServe(gateway, mcpRequest("tools/list", nil, "2026-07-28"))
	require.Contains(t, response.Body.String(), "syncing")
	require.Equal(t, "1", response.Header().Get("Retry-After"))
	close(watch.release)
	require.Eventually(t, func() bool { return !gateway.desc.Snapshot().Syncing }, time.Second, time.Millisecond)
	response = mcpServe(gateway, mcpRequest("tools/list", nil, "2026-07-28"))
	require.Contains(t, response.Body.String(), "schema is unavailable")
	client.Publish(hubwatch.Event{Kind: hubwatch.EventKindUpsert, Key: watched.FormatDescriptorDataKey("demo.Missing"), Value: vcode.MustMarshalJsonS(skeldesc.Data{SkelName: "demo.Missing"})})
	require.Eventually(t, func() bool { s := gateway.desc.Snapshot(); return !s.Syncing && s.Data["demo.Missing"] != nil }, time.Second, time.Millisecond)
	var result mcp.ListToolsResult
	mcpResult(t, mcpServe(gateway, mcpRequest("tools/list", nil, "2026-07-28")), &result)
	require.Len(t, result.Tools, 1)
}
