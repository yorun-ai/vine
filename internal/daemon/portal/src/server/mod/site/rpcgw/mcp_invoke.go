package rpcgw

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strings"

	validator "github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

func (g *RpcGateway) callMcpTool(_ context.Context, httpCtx *spec.Context, service string, method string, tool *mcp.Tool, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := resolveMcpSchema(tool.InputSchema)
	if err != nil {
		return nil, &jsonrpc.Error{Code: -32000, Message: "API input schema is unavailable"}
	}
	output, err := resolveMcpSchema(tool.OutputSchema)
	if err != nil {
		return nil, &jsonrpc.Error{Code: -32000, Message: "API output schema is unavailable"}
	}
	arguments := []byte(call.Params.Arguments)
	if len(arguments) == 0 {
		arguments = []byte("{}")
	}
	if err := validateMcpValue(input, arguments); err != nil {
		return mcpToolFailure(ex.InvalidRequest, "invalid tool arguments: "+err.Error(), ""), nil
	}
	body, err := vrpchttp.EncodeJSONRequest(arguments)
	if err != nil {
		return mcpToolFailure(ex.InvalidRequest, "invalid tool arguments", ""), nil
	}
	// Once accepted, business execution follows the gateway's timeout and shutdown,
	// not HTTP disconnects or SDK cancellation of the client-facing call.
	invokeCtx := httpCtx.Request.Context()
	request, err := vrpchttp.NewRequest(invokeCtx, "http://portal", "/"+service+"/"+method, body, rpchttp.ContentTypeJson, rpchttp.ContentTypeJson)
	if err != nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: "cannot prepare tool invocation"}
	}
	// Only credentials cross the transport boundary. Actor/initiator/options and
	// other client-supplied vRPC headers must never become trusted backend context.
	request.Header["Authorization"] = append([]string(nil), httpCtx.Request.Header.Values("Authorization")...)
	request.Header.Set("User-Agent", httpCtx.Request.UserAgent())
	rpchttp.EncodeClientToHeader(request.Header, g.app)
	trace, err := rpchttp.DecodeTraceFromHeader(httpCtx.Request.Header)
	if err != nil {
		trace, _ = meta.NewTrace(httpCtx.ResponseWriter.Header().Get(spec.HeaderPortalTraceId), "")
	}
	trace = trace.NewChildTrace()
	rpchttp.EncodeTraceToHeader(request.Header, trace)
	dialer := httpCtx.Request.UserAgent()
	if client := call.ClientInfo(); client != nil {
		dialer = strings.TrimSpace(client.Name + "/" + client.Version + " " + dialer)
	}
	initiator, err := meta.NewInitiator("vine.mcp", "0.0.0", meta.MustNewInstanceId(), dialer, httpCtx.RemoteAddr)
	if err != nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: "cannot prepare tool initiator"}
	}
	request.Header.Set(rpchttp.HeaderRpcInitiator, meta.EncodeInitiatorToBase64(initiator))
	response := &_McpInvokeResponse{header: make(http.Header)}
	g.invoke(&spec.Context{Request: request, ResponseWriter: response, RemoteAddr: httpCtx.RemoteAddr}, trace, initiator, service, method)
	return mcpInvokeResult(response, output), nil
}

func resolveMcpSchema(value any) (*validator.Resolved, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var schema validator.Schema
	if err := json.Unmarshal(body, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
}

func validateMcpValue(schema *validator.Resolved, body []byte) error {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	return schema.Validate(value)
}

// _McpInvokeResponse captures the bounded, decompressed vRPC envelope. No backend
// response headers or credentials are copied into the MCP HTTP response.
type _McpInvokeResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (r *_McpInvokeResponse) Header() http.Header { return r.header }
func (r *_McpInvokeResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *_McpInvokeResponse) Write(body []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	return r.body.Write(body)
}

func mcpInvokeResult(response *_McpInvokeResponse, output *validator.Resolved) *mcp.CallToolResult {
	invalid := func() *mcp.CallToolResult {
		return mcpToolFailure(ex.UnexpectedResponse, "invalid API response", "")
	}
	status, err := rpchttp.DecodeStatusCodeFromHeader(response.header)
	if err != nil || response.status < 200 || response.status >= 300 || vrpchttp.MediaTypeOf(response.header.Get("Content-Type")) != rpchttp.ContentTypeJson {
		return invalid()
	}
	payload, err := vrpchttp.DecodeResponse(response.body.Bytes(), response.header.Get("Content-Type"))
	if err != nil {
		return invalid()
	}
	failure, err := payload.DecodeError()
	if err != nil || (status == ex.OK) != (failure == nil) {
		return invalid()
	}
	if failure != nil {
		return mcpToolFailure(status, failure.Message, failure.Reason)
	}
	// Preserve raw JSON numbers (notably int64) instead of re-encoding float64.
	structured := jsontext.Value(payload.ResultBytes)
	if validateMcpValue(output, structured) != nil {
		return invalid()
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(structured)}},
		StructuredContent: structured,
	}
}

func mcpToolFailure(code ex.Code, message string, reason string) *mcp.CallToolResult {
	text := fmt.Sprintf("%s: %s", code, message)
	if reason != "" {
		text += " (" + reason + ")"
	}
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
