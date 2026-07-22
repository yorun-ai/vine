package server

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
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
	routes       []*spec.Route
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

func (s *Server) Routes() []spec.RouteInfo {
	routes := make([]spec.RouteInfo, 0, len(s.routes))
	for _, route := range s.routes {
		routes = append(routes, route)
	}
	return routes
}

func (s *Server) WebInfos() []spec.WebInfo {
	return vslice.Clone(s.webInfos)
}

func (s *Server) initRoutes() {
	routers := vslice.Map(s.handlerTypes, func(handlerType reflect.Type) *spec.Router {
		checkHandlerType(handlerType)
		webInfo := spec.GetWebInfo(handlerType)
		vpre.Check(handlerType.Implements(webInfo.ServerType()), "web handler type %s must implement web server %s", handlerType, webInfo.ServerType())
		if !vslice.Contains(s.webInfos, webInfo) {
			s.webInfos = append(s.webInfos, webInfo)
		}
		router := spec.NewRouter(handlerType, "/"+webInfo.SkelName())
		handlerIns := reflect.New(handlerType.Elem()).Interface()
		handlerIns.(spec.Handler).Routes(router)
		return router
	})
	routers = flattenRouters(routers)

	for _, router := range routers {
		for _, route := range router.Routes() {
			s.routes = append(s.routes, route.WithBasePath(router.BasePath()))
		}
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

	for _, route := range s.routes {
		s.ginHandler(route)
	}
}

func (s *Server) ginHandler(route *spec.Route) {
	s.ginEngine.Handle(route.Method(), route.Path(), func(ginCtx *gin.Context) {
		s.executor.Execute(route, ginCtx)
	})
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
		stack := ex.PanicStack(err)
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

func flattenRouters(routers []*spec.Router) []*spec.Router {
	flatten := vslice.Clone(routers)
	for _, r := range routers {
		flatten = append(flatten, flattenRouters(r.SubRouters())...)
	}
	return flatten
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
