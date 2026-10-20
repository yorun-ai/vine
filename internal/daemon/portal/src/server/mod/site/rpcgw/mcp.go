package rpcgw

import (
	"context"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/skel/jsonschema"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/util/gwutil"
	vrpchttp "go.yorun.ai/vrpc/transport/http"
)

const (
	mcpPageSize        = 100
	mcpProtocolVersion = "2026-07-28"
)

var mcpToolNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

func (g *RpcGateway) serveMcp(ctx *spec.Context, cors watched.PortalCors) {
	w, r := ctx.ResponseWriter, ctx.Request
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set(spec.HeaderPortalTraceId, rpcTraceIdOrNew(r.Header))
	g.mutex.Lock()
	enabled := g.mcpEnabled
	g.mutex.Unlock()
	if !enabled || r.URL.Path != pathMcp {
		http.NotFound(w, r)
		return
	}
	if !spec.AllowRequestOrigin(r, cors, ctx.EntryOrigin) {
		http.Error(w, "origin is not allowed", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodOptions {
		spec.ServeOptions(w, r, cors, ctx.EntryOrigin, []string{http.MethodPost})
		return
	}
	if spec.ApplyCORS(w, r, cors, ctx.EntryOrigin) {
		for _, header := range []string{spec.HeaderPortalTraceId, "MCP-Protocol-Version", "Retry-After"} {
			spec.ExposeHeader(w.Header(), header)
		}
	}
	// The SDK still accepts headerless legacy initialization even when its
	// advertised versions are restricted. Require the modern version at ingress.
	if version := r.Header.Get("MCP-Protocol-Version"); r.Method == http.MethodPost && version != mcpProtocolVersion {
		data, _ := json.Marshal(mcp.UnsupportedProtocolVersionData{
			Supported: []string{mcpProtocolVersion}, Requested: version,
		})
		failure := &jsonrpc.Error{
			Code: mcp.CodeUnsupportedProtocolVersion, Message: "Unsupported protocol version", Data: data,
		}
		if version == "" {
			failure.Code = mcp.CodeHeaderMismatch
			failure.Message = "MCP-Protocol-Version header is required; supported version: " + mcpProtocolVersion
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.MarshalWrite(w, map[string]any{"jsonrpc": "2.0", "id": nil, "error": failure})
		return
	}
	requestCtx, cancel := gwutil.ContextWithoutClientCancel(r.Context(), g.context, maxRpcTimeout)
	defer cancel()
	next := *ctx
	next.Request = r.WithContext(requestCtx)
	// Servers are request-scoped: only the requested catalog or method is
	// materialized. The descriptor manager remains the sole subscription/cache.
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return g.newMcpServer(&next)
	}, &mcp.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, PropagateRequestCancellation: true,
		MaxRequestBodyBytes: vrpchttp.MaxRequestBodyBytes,
	})
	handler.ServeHTTP(&_McpResponseWriter{ResponseWriter: w}, next.Request)
}

// The SDK supplies transport cache headers; API results must never be stored.
type _McpResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *_McpResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *_McpResponseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.Header().Set("Cache-Control", "no-store")
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *_McpResponseWriter) Write(body []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.ResponseWriter.Write(body)
}

func (g *RpcGateway) newMcpServer(httpCtx *spec.Context) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "vine.portal", Version: g.app.Version()}, &mcp.ServerOptions{
		PageSize:                  mcpPageSize,
		SupportedProtocolVersions: []string{mcpProtocolVersion},
		Capabilities:              &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
		SetCacheable: func(_ context.Context, _ mcp.Request, value *mcp.Cacheable) {
			*value = mcp.Cacheable{TTLMs: 0, CacheScope: "private"}
		},
	})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			timeout, err := mcpOptionsTimeout(request.GetParams().GetMeta())
			if err != nil {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error()}
			}
			// Count decoding and admission against the same budget as execution.
			limit, _ := httpCtx.Request.Context().Deadline()
			requestCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), limit.Add(timeout-maxRpcTimeout))
			defer cancel()
			stop := context.AfterFunc(httpCtx.Request.Context(), cancel)
			defer stop()
			nextHTTP := *httpCtx
			nextHTTP.Request = httpCtx.Request.WithContext(requestCtx)
			if method != "tools/list" && method != "tools/call" {
				return next(requestCtx, method, request)
			}
			snapshot, services := g.apiSnapshot()
			if snapshot.Syncing {
				httpCtx.ResponseWriter.Header().Set("Retry-After", "1")
				return nil, &jsonrpc.Error{Code: -32000, Message: "API descriptors are syncing"}
			}
			requested := ""
			if method == "tools/call" {
				requested = request.(*mcp.CallToolRequest).Params.Name
			}
			for _, service := range services {
				for _, operation := range service.Methods {
					name := mcpToolName(service.SkelName, operation.SkelName)
					if method == "tools/call" && name != requested {
						continue
					}
					input, output, err := jsonschema.Method(operation, snapshot.Data, snapshot.Enums)
					if err != nil {
						rpcGatewayLogger.Warn("vine.portal MCP schema is unavailable", "service", service.SkelName, "method", operation.SkelName, "error", err)
						return nil, &jsonrpc.Error{Code: -32000, Message: "API schema is unavailable"}
					}
					tool := &mcp.Tool{
						Name: name, Title: service.SkelName + "/" + operation.SkelName,
						Description: mcpToolDescription(service, operation),
						InputSchema: input, OutputSchema: output,
					}
					server.AddTool(tool, func(ctx context.Context, call *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
						return g.callMcpTool(ctx, &nextHTTP, service.SkelName, operation.SkelName, tool, call)
					})
				}
			}
			return next(requestCtx, method, request)
		}
	})
	return server
}

func mcpToolName(service string, method string) string {
	name := service + "." + method
	if mcpToolNamePattern.MatchString(name) {
		return name
	}
	// Keep unusual/long wire names addressable without ambiguous replacement.
	// Normal names always contain a dot, so they cannot collide with this form.
	return fmt.Sprintf("vine_%x", sha256.Sum256([]byte(name)))
}

func mcpToolDescription(service *skeldesc.Service, method *skeldesc.Method) string {
	parts := []string{service.Description, method.Description}
	if service.Deprecated || method.Deprecated {
		parts = append(parts, "Deprecated.", service.DeprecatedReason, method.DeprecatedReason)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// mcpOptionsTimeout uses the same wire codec and limits as the vRPC ingress.
func mcpOptionsTimeout(metadata map[string]any) (time.Duration, error) {
	header := make(http.Header)
	if value, present := metadata["vmcp-options"]; present {
		options, ok := value.(string)
		if !ok {
			return 0, fmt.Errorf("invalid vmcp-options: expected a string")
		}
		header.Set("vrpc-options", options)
	}
	_, timeout, err := decodeRpcOptionsTimeout(header)
	if err != nil {
		return 0, fmt.Errorf("invalid vmcp-options: %s", strings.ReplaceAll(err.Error(), "vrpc-options", "vmcp-options"))
	}
	return timeout, nil
}
