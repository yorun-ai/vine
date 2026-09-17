package web

import (
	"io/fs"
	"reflect"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/core/di"
	"go.yorun.ai/vine/core/meta"
	internalassets "go.yorun.ai/vine/internal/core/web/assets"
	internaldevproxy "go.yorun.ai/vine/internal/core/web/devproxy"
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

// AssetsServer serves files from an AssetsAccessor. Embed it by value in a Web
// handler, set its accessor in DIInit, and delegate Routes to AssetsServer.Routes.
// Its GinCtx is injected for each execution; Serve is promoted to the Web handler.
type AssetsServer = internalassets.Server

// DevProxyServer serves a Web from the development server named by a state file,
// so a development run serves the frontend from its sources while the rest of the
// Web handler stays the generated one. Embed it by value in the Web handler of a
// development build, set the state file in DIInit, and delegate Routes to
// DevProxyServer.Routes.
type DevProxyServer = internaldevproxy.Server

// NewContext creates a Web execution context from Gin and Vine metadata.
func NewContext(ginCtx *gin.Context, route Route, trace meta.Trace, initiator meta.Initiator, actor meta.Actor) Context {
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
