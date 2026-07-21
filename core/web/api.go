package web

import (
	"io/fs"
	"reflect"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/meta"
	internalassets "go.yorun.ai/vine/internal/core/web/assets"
	internalproxy "go.yorun.ai/vine/internal/core/web/proxy"
	internalserver "go.yorun.ai/vine/internal/core/web/server"
	internalspec "go.yorun.ai/vine/internal/core/web/spec"
)

// Context carries Gin and Vine metadata for one Web request.
type Context = internalspec.Context

// Executor invokes a registered Web handler.
type Executor = internalserver.Executor

// HandleFunc handles one registered Web route.
type HandleFunc = internalspec.HandleFunc

// Handler groups route handling behavior generated from a Web contract.
type Handler = internalspec.Handler

// Option configures a Web server.
type Option = internalserver.Option

// Route describes a generated Web route.
type Route = internalspec.Route

// RouteInfo is runtime metadata for a registered route.
type RouteInfo = internalspec.RouteInfo

// Router registers route handlers.
type Router = internalspec.Router

// Server receives Web requests and dispatches them to an Executor.
type Server = internalserver.Server

// WebInfo is runtime metadata derived from a WebSpec.
type WebInfo = internalspec.WebInfo

// WebSpec describes a generated Web contract.
type WebSpec = internalspec.WebSpec

// AssetsAccessor opens files from an embedded or archived asset set.
type AssetsAccessor = internalassets.Accessor

// AssetsServer serves files from an AssetsAccessor.
type AssetsServer = internalassets.Server

// ProxyOption configures a reverse proxy.
type ProxyOption = internalproxy.Option

// ReverseProxy forwards Web requests to an upstream server.
type ReverseProxy = internalproxy.ReverseProxy

// NewContext creates a Web execution context from Gin and Vine metadata.
func NewContext(ginCtx *gin.Context, route RouteInfo, trace meta.Trace, initiator meta.Initiator, actor meta.Actor) Context {
	return internalspec.NewContext(ginCtx, route, trace, initiator, actor)
}

// NewServer creates a Web server from opt.
func NewServer(opt Option) *Server {
	return internalserver.NewServer(opt)
}

// NewContainerExecutor creates an Executor backed by a DI container and filter chain.
func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return internalserver.NewContainerExecutor(filterTypes, bindAppliers)
}

// NewAssetsServer creates a server for accessor.
func NewAssetsServer(accessor AssetsAccessor) AssetsServer {
	return internalassets.NewServer(accessor)
}

// NewReverseProxy creates a reverse proxy from opt.
func NewReverseProxy(opt ProxyOption) *ReverseProxy {
	return internalproxy.NewReverseProxy(opt)
}

// NewEmbedAssetsAccessor creates an accessor rooted at root within fsys.
func NewEmbedAssetsAccessor(fsys fs.FS, root string) AssetsAccessor {
	return internalassets.NewEmbedAccessor(fsys, root)
}

// NewTarZstAssetsAccessor creates an accessor for Zstandard-compressed TAR content.
func NewTarZstAssetsAccessor(content []byte) AssetsAccessor {
	return internalassets.NewTarZstAccessor(content)
}

// NewTarGzipAssetsAccessor creates an accessor for gzip-compressed TAR content.
func NewTarGzipAssetsAccessor(content []byte) AssetsAccessor {
	return internalassets.NewTarGzipAccessor(content)
}

// NewZipAssetsAccessor creates an accessor for ZIP content.
func NewZipAssetsAccessor(content []byte) AssetsAccessor {
	return internalassets.NewZipAccessor(content)
}

// Register adds webSpec to the process-wide Web registry.
func Register(webSpec *WebSpec) {
	internalspec.Register(webSpec)
}

// RegisteredWebInfos returns metadata for all registered Web contracts.
func RegisteredWebInfos() []WebInfo {
	return internalspec.RegisteredWebInfos()
}
