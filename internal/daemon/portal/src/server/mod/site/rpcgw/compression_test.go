package rpcgw

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vcode"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

var excludedRpcHeaders = []string{"Cookie", "Origin", "X-Forwarded-For", "Forwarded", "X-Custom", "Mcp-Name", "MCP-Protocol-Version", "vrpc-status", "vrpc-server", "Connection", "Trailer"}

func TestPrepareForwardRequestAllowlist(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	request := httptest.NewRequest("POST", "http://portal/invoke/a/b", bytes.NewBufferString("request")).WithContext(ctx)
	setTestAuthHeaders(request)
	request.Header.Set("Accept", rpchttp.ContentTypeCbor)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("User-Agent", "client/1")
	request.Header.Set("Authorization", "secret")
	request.Header.Set("vrpc-options", "timeout=120s,destination=private")
	request.Header.Set("vrpc-actor", "admitted-actor")
	request.Header.Set("vrpc-initiator", "portal-initiator")
	for _, name := range excludedRpcHeaders {
		request.Header.Set(name, "untrusted")
	}
	request.Trailer = http.Header{"X-Secret": []string{"secret"}}
	original := request.Header.Clone()
	encoding, forwarded := prepareForwardRequest(request)
	require.Equal(t, "gzip", encoding)
	require.Equal(t, original, request.Header)
	require.Equal(t, request.Context(), forwarded.Context())
	require.True(t, request.Body == forwarded.Body)
	require.Nil(t, forwarded.Trailer)
	for _, name := range append(excludedRpcHeaders, "User-Agent", "Authorization", "Accept-Encoding", "Content-Encoding") {
		require.Empty(t, forwarded.Header.Get(name), name)
	}
	require.Equal(t, rpchttp.ContentTypeCbor, forwarded.Header.Get("Accept"))
	require.Equal(t, "admitted-actor", forwarded.Header.Get("vrpc-actor"))
	require.Equal(t, "portal-initiator", forwarded.Header.Get("vrpc-initiator"))
	options, err := rpchttp.DecodeOptionsFromHeader(forwarded.Header)
	require.NoError(t, err)
	require.Positive(t, options.Timeout)
	require.LessOrEqual(t, options.Timeout, 10*time.Second)
	oldTrace, err := rpchttp.DecodeTraceFromHeader(request.Header)
	require.NoError(t, err)
	trace, err := rpchttp.DecodeTraceFromHeader(forwarded.Header)
	require.NoError(t, err)
	require.Equal(t, oldTrace.Id(), trace.Id())
	require.NotEqual(t, oldTrace.Span(), trace.Span())
	forwarded.Header.Set("Accept", "changed")
	require.Equal(t, original, request.Header)
}

func TestRpcgwHeaderContractAcrossTransports(t *testing.T) {
	for _, network := range []bool{false, true} {
		name := "inproc"
		if network {
			name = "http2"
		}
		t.Run(name, func(t *testing.T) {
			for _, format := range []string{rpchttp.ContentTypeJson, rpchttp.ContentTypeCbor, "mcp"} {
				t.Run(format, func(t *testing.T) {
					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if network {
							require.Equal(t, 2, r.ProtoMajor)
						}
						for _, name := range append(excludedRpcHeaders, "Authorization", "User-Agent", "Content-Encoding", "Accept-Encoding") {
							require.Empty(t, r.Header.Get(name), name)
						}
						actor, err := meta.DecodeActorFromBase64(r.Header.Get(rpchttp.HeaderRpcActor))
						require.NoError(t, err)
						require.True(t, actor.IsAnonymous())
						initiator, err := meta.DecodeInitiatorFromBase64(r.Header.Get(rpchttp.HeaderRpcInitiator))
						require.NoError(t, err)
						require.Contains(t, initiator.Dialer(), "test-client/1")
						require.Equal(t, "192.0.2.1", initiator.IpAddr())
						client, err := rpchttp.DecodeClientFromHeader(r.Header)
						require.NoError(t, err)
						wantClient := "demo.client"
						contentType := format
						if format == "mcp" {
							wantClient = "vine.portal"
							contentType = rpchttp.ContentTypeJson
						}
						require.Equal(t, wantClient, client.Name())
						require.Equal(t, contentType, r.Header.Get("Content-Type"))
						require.Equal(t, contentType, r.Header.Get("Accept"))
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						var args map[string]any
						require.NoError(t, vrpchttp.DecodeRequest(body, &args, contentType))
						options, err := rpchttp.DecodeOptionsFromHeader(r.Header)
						require.NoError(t, err)
						require.Positive(t, options.Timeout)
						require.LessOrEqual(t, options.Timeout, 30*time.Second)
						payload, err := vrpchttp.EncodeResponse(nil, nil, contentType)
						require.NoError(t, err)
						w.Header().Set("Content-Type", contentType)
						w.Header().Set("vrpc-status", "OK")
						rpchttp.EncodeServerToHeader(w.Header(), testServerApp())
						_, _ = w.Write(payload)
					})
					endpoint := "link+inproc://vine/header-contract-" + meta.NewId()
					if network {
						server := httptest.NewUnstartedServer(handler)
						server.Config.Protocols = new(http.Protocols)
						server.Config.Protocols.SetUnencryptedHTTP2(true)
						server.Start()
						defer server.Close()
						endpoint = server.URL
					} else {
						registerTestIngress(t, endpoint, handler)
					}
					watch := newTestDescriptorWatch(t)
					watch.SetValue(watched.FormatRpcServiceRegistrationKey("demo.UserService", "demo.app", "instance-1"), vcode.MustMarshalJsonS(watched.RpcServiceRegistration{Endpoint: endpoint, ServiceName: "demo.UserService", AppName: "demo.app", AppInstanceId: "instance-1"}))
					gateway := newInspectGateway(t, watch, mcpConfig("demo.UserService"))
					var request *http.Request
					if format == "mcp" {
						request = mcpRequest("tools/call", map[string]any{"name": "demo.UserService.Get"}, mcpProtocolVersion)
					} else {
						body, err := vrpchttp.EncodeRequest(map[string]any{}, format)
						require.NoError(t, err)
						request = httptest.NewRequest("POST", "http://portal/invoke/demo.UserService/Get", bytes.NewReader(body))
						setTestAuthHeaders(request)
						request.Header.Set("Content-Type", format)
						request.Header.Set("Accept", format)
					}
					for _, name := range excludedRpcHeaders {
						if name != "Mcp-Name" && name != "MCP-Protocol-Version" && name != "Origin" {
							request.Header.Set(name, "untrusted")
						}
					}
					request.Header.Set("User-Agent", "test-client/1")
					request.Header.Set("vrpc-actor", "spoofed")
					request.Header.Set("vrpc-initiator", "spoofed")
					request.Header.Set("Content-Encoding", "identity")
					response := mcpServe(gateway, request)
					require.Equal(t, http.StatusOK, response.Code, response.Body.String())
					if format == "mcp" {
						var result mcp.CallToolResult
						mcpResult(t, response, &result)
						require.False(t, result.IsError, result.Content)
					} else {
						require.Equal(t, format, response.Header().Get("Content-Type"))
					}
				})
			}
		})
	}
}

func TestRpcgwRejectsUnsupportedRequestEncoding(t *testing.T) {
	for _, values := range [][]string{{"gzip"}, {"br"}, {"identity, gzip"}, {"identity", "identity"}, {""}} {
		for _, mcpCall := range []bool{false, true} {
			gateway := newInspectGateway(t, newTestDescriptorWatch(t), mcpConfig("demo.UserService"))
			request := mcpRequest("tools/call", map[string]any{"name": "demo.UserService.Get"}, mcpProtocolVersion)
			if !mcpCall {
				request = httptest.NewRequest("POST", "http://portal/invoke/demo.UserService/Get", bytes.NewBufferString("{}"))
				setTestAuthHeaders(request)
			}
			request.Header["Content-Encoding"] = values
			response := mcpServe(gateway, request)
			if mcpCall {
				require.Equal(t, http.StatusUnsupportedMediaType, response.Code)
			} else {
				require.Equal(t, string(ex.InvalidRequest), response.Header().Get("vrpc-status"))
			}
			require.Contains(t, response.Body.String(), "Content-Encoding")
		}
	}
}
