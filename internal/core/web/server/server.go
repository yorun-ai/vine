package server

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/web/spec"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

type Option struct {
	HandlerTypes []reflect.Type
	Executor     Executor
}

type Server struct {
	opt *Option

	handlerTypes []reflect.Type
	webInfos     []spec.WebInfo
	routes       []spec.Route
	executor     Executor

	ginEngine *gin.Engine
}

func NewServer(opt Option) *Server {
	s := &Server{opt: &opt}
	s.init()
	return s
}

func (s *Server) init() {
	s.handlerTypes = vslice.Clone(s.opt.HandlerTypes)
	s.initRoutes()
	s.initExecutor()
	s.initEngine()
}

func (s *Server) HTTPHandler() http.Handler {
	return s.ginEngine
}

func (s *Server) Routes() []spec.Route {
	routes := make([]spec.Route, 0, len(s.routes))
	for _, route := range s.routes {
		routes = append(routes, route)
	}
	return routes
}

func (s *Server) WebInfos() []spec.WebInfo {
	return vslice.Clone(s.webInfos)
}

func (s *Server) initRoutes() {
	for _, handlerType := range s.handlerTypes {
		checkHandlerType(handlerType)
		webInfo := spec.GetWebInfo(handlerType)
		vpre.Check(handlerType.Implements(webInfo.ServerType()), "web handler type %s must implement web server %s", handlerType, webInfo.ServerType())
		if !vslice.Contains(s.webInfos, webInfo) {
			s.webInfos = append(s.webInfos, webInfo)
		}
		router := spec.NewRouter(handlerType, webInfo.MountPath())
		handlerIns := reflect.New(handlerType.Elem()).Interface()
		handlerIns.(spec.Handler).Routes(router)
		prefix := "/" + webInfo.SkelName()
		s.routes = append(s.routes, spec.CollectRoutes(router, prefix)...)
	}
}

func checkHandlerType(handlerType reflect.Type) {
	vpre.Check(handlerType.Kind() == reflect.Pointer && handlerType.Elem().Kind() == reflect.Struct, "web handler type %s must be a pointer to struct", handlerType)
}

func (s *Server) initExecutor() {
	s.executor = s.opt.Executor
	if s.executor == nil {
		s.executor = NewContainerExecutor(nil, nil)
	}
	s.executor.Init(s.Routes())
}

func (s *Server) initEngine() {
	gin.SetMode(gin.ReleaseMode)
	s.ginEngine = gin.New()
	s.ginEngine.Use(s.ginLogger(), s.ginRecovery())

	restoreGinRoutePrinter := s.overrideGinDebugRoutePrinter()
	defer restoreGinRoutePrinter()

	// Every Web owns a Gin group under its own name, so whatever Vine wires for one
	// Web stays with the routes of that Web. All groups share the engine, and with
	// it the logging and recovery chain above.
	groupByHandlerType := map[reflect.Type]*gin.RouterGroup{}
	for _, route := range s.routes {
		s.ginHandler(s.webGroup(groupByHandlerType, route), route)
	}
}

// webGroup returns the Gin group of a Web, creating it on first use.
func (s *Server) webGroup(groups map[reflect.Type]*gin.RouterGroup, route spec.Route) *gin.RouterGroup {
	handlerType := route.HandlerType()
	if group, ok := groups[handlerType]; ok {
		return group
	}
	basePath := webGroupPath(spec.GetWebInfo(handlerType).SkelName())
	group := s.ginEngine.Group(basePath, stripWebName(basePath))
	groups[handlerType] = group
	return group
}

// webGroupPath is the route prefix one Web owns: the Web name routes carry.
func webGroupPath(skelName string) string {
	return "/" + skelName
}

// stripWebName drops the Web name from the path a Web handler reads. The name
// belongs to the dispatch Link and the application share, so a handler sees the
// path its Web serves: the mount when the Web declares one, and what follows it.
func stripWebName(basePath string) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		request := ginCtx.Request
		path := strings.TrimPrefix(request.URL.Path, basePath)
		if path == "" {
			path = "/"
		}
		request.URL.Path = path
		if request.URL.RawPath != "" {
			// Keep the escaped suffix only when the dispatch prefix matches too.
			// Otherwise let URL reconstruct a valid encoding from Path.
			rawPath, ok := strings.CutPrefix(request.URL.RawPath, basePath)
			request.URL.RawPath = ""
			if ok {
				request.URL.RawPath = rawPath
			}
		}
		request.RequestURI = request.URL.RequestURI()
	}
}

func (s *Server) ginHandler(group *gin.RouterGroup, route spec.Route) {
	group.Handle(route.Method(), groupRelativePath(group.BasePath(), route), func(ginCtx *gin.Context) {
		s.executor.Execute(route, ginCtx)
	})
}

// groupRelativePath is the route path inside the Web group: the collected path
// without the Web name the group already carries.
func groupRelativePath(basePath string, route spec.Route) string {
	vpre.Check(route.Path() == basePath || strings.HasPrefix(route.Path(), basePath+"/"),
		"web route %s %s is outside the group %s of its Web", route.Method(), route.Path(), basePath)
	path := strings.TrimPrefix(route.Path(), basePath)
	if path == "" {
		return "/"
	}
	return path
}

func (s *Server) ginLogger() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		start := time.Now()
		ginCtx.Next()

		attrs := []any{
			"method", ginCtx.Request.Method,
			"path", ginCtx.Request.URL.Path,
			"status", ginCtx.Writer.Status(),
			"duration", time.Since(start),
			"clientIp", ginCtx.ClientIP(),
		}
		if rawQuery := ginCtx.Request.URL.RawQuery; rawQuery != "" {
			attrs = append(attrs, "query", rawQuery)
		}

		switch {
		case len(ginCtx.Errors) > 0:
			logger.Warn("web request finished", append(attrs, "errors", ginCtx.Errors.String())...)
		case ginCtx.Writer.Status() >= http.StatusInternalServerError:
			logger.Error("web request finished", attrs...)
		case ginCtx.Writer.Status() >= http.StatusBadRequest:
			logger.Warn("web request finished", attrs...)
		default:
			logger.Debug("web request finished", attrs...)
		}
	}
}

func (s *Server) ginRecovery() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if err, ok := recovered.(ex.Error); ok {
					s.recoverWebError(ginCtx, err)
					return
				}
				if isAbortHandlerPanic(recovered) {
					ginCtx.Abort()
					return
				}
				logger.Error("web request panic recovered", "panic", recovered, "stack", string(debug.Stack()))
				ginCtx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		ginCtx.Next()
	}
}

func (s *Server) recoverWebError(ginCtx *gin.Context, err ex.Error) {
	if err.Type() == ex.SystemError {
		stack := ex.Stack(err)
		if stack == "" {
			stack = string(debug.Stack())
		}
		logger.Error("web request system error recovered",
			"error", err,
			"stack", stack,
			"method", ginCtx.Request.Method,
			"path", ginCtx.Request.URL.Path,
		)
	}

	if ginCtx.Writer.Written() {
		if err.Type() == ex.ApplicationError {
			logger.Warn("web request application error recovered after response started",
				"error", err,
				"method", ginCtx.Request.Method,
				"path", ginCtx.Request.URL.Path,
			)
		}
		ginCtx.Abort()
		return
	}

	ginCtx.AbortWithStatus(ex.HTTPStatusCode(err.Code()))
}

func isAbortHandlerPanic(err any) bool {
	if errValue, ok := err.(error); ok {
		return errors.Is(errValue, http.ErrAbortHandler)
	}
	return false
}

func (s *Server) overrideGinDebugRoutePrinter() func() {
	if !gin.IsDebugging() {
		return func() {}
	}

	handlerNames := map[string]string{}
	for _, route := range s.routes {
		handlerNames[routeKey(route.Method(), route.Path())] = route.HandlerName()
	}

	previous := gin.DebugPrintRouteFunc
	gin.DebugPrintRouteFunc = func(httpMethod string, absolutePath string, handlerName string, nuHandlers int) {
		if realHandlerName, ok := handlerNames[routeKey(httpMethod, absolutePath)]; ok {
			handlerName = realHandlerName
		}

		if previous != nil {
			previous(httpMethod, absolutePath, handlerName, nuHandlers)
			return
		}

		_, _ = fmt.Fprintf(gin.DefaultWriter, "[GIN-debug] %-6s %-25s --> %s (%d handlers)\n",
			httpMethod,
			absolutePath,
			handlerName,
			nuHandlers,
		)
	}
	return func() {
		gin.DebugPrintRouteFunc = previous
	}
}

func routeKey(method string, path string) string {
	return method + " " + path
}
