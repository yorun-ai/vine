package spec

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

type Route interface {
	Method() string
	Path() string
	HandlerType() reflect.Type
	HandlerMethod() reflect.Method
	HandlerName() string
}

type _Route struct {
	sourceMethod  string
	sourcePath    string
	handlerType   reflect.Type
	handlerMethod reflect.Method
}

func (r *_Route) Method() string {
	return r.sourceMethod
}

func (r *_Route) Path() string {
	return r.sourcePath
}

func (r *_Route) HandlerType() reflect.Type {
	return r.handlerType
}

func (r *_Route) HandlerMethod() reflect.Method {
	return r.handlerMethod
}

func (r *_Route) HandlerName() string {
	return runtime.FuncForPC(r.handlerMethod.Func.Pointer()).Name()
}

type Router struct {
	handlerType reflect.Type
	basePath    string
	routes      []*_Route
	subRouters  map[string]*Router
}

func NewRouter(handlerType reflect.Type) *Router {
	return &Router{
		handlerType: handlerType,
		routes:      []*_Route{},
		subRouters:  map[string]*Router{},
	}
}

// BasePath returns the router's accumulated mount path within the Web.
// The root router returns an empty string. Internal dispatch prefixes and
// entry prefixes stripped before forwarding are not included.
func (r *Router) BasePath() string {
	return r.basePath
}

func (r *Router) Routes() []Route {
	routes := make([]Route, 0, len(r.routes))
	for _, route := range r.routes {
		routes = append(routes, route)
	}
	return routes
}

func (r *Router) SubRouters() []*Router {
	routers := make([]*Router, 0, len(r.subRouters))
	for _, subRouter := range r.subRouters {
		routers = append(routers, subRouter)
	}
	return routers
}

func (r *Router) SubRouter(path string) *Router {
	if _, exists := r.subRouters[path]; !exists {
		r.subRouters[path] = &Router{
			handlerType: r.handlerType,
			basePath:    r.basePath + path,
			routes:      []*_Route{},
			subRouters:  map[string]*Router{},
		}
	}
	return r.subRouters[path]
}

func (r *Router) Handle(method string, path string, handleFunc HandleFunc) {
	targetMethodName := r.methodName(handleFunc)
	targetMethod, ok := r.handlerType.MethodByName(targetMethodName)
	if !ok {
		panic(fmt.Sprintf("method=%s not found in type=%s", targetMethodName, r.handlerType.Name()))
	}

	r.routes = append(r.routes, &_Route{
		sourceMethod:  method,
		sourcePath:    path,
		handlerType:   r.handlerType,
		handlerMethod: targetMethod,
	})
}

func (r *Router) methodName(handleFunc HandleFunc) string {
	name := runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()
	parts := strings.Split(name, ".")
	name = parts[len(parts)-1]
	parts = strings.Split(name, "-")
	name = parts[0]
	return name
}

func (r *Router) ANY(path string, handleFunc HandleFunc) {
	r.GET(path, handleFunc)
	r.POST(path, handleFunc)
	r.PUT(path, handleFunc)
	r.PATCH(path, handleFunc)
	r.DELETE(path, handleFunc)
	r.OPTIONS(path, handleFunc)
	r.HEAD(path, handleFunc)
}

func (r *Router) POST(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodPost, path, handleFunc)
}

func (r *Router) GET(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodGet, path, handleFunc)
}

func (r *Router) DELETE(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodDelete, path, handleFunc)
}

func (r *Router) PATCH(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodPatch, path, handleFunc)
}

func (r *Router) PUT(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodPut, path, handleFunc)
}

func (r *Router) OPTIONS(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodOptions, path, handleFunc)
}

func (r *Router) HEAD(path string, handleFunc HandleFunc) {
	r.Handle(http.MethodHead, path, handleFunc)
}

// CollectRoutes assembles server routes with the Web dispatch prefix and each
// router's accumulated base path. It is not exposed by the public web facade.
func CollectRoutes(r *Router, prefix string) []Route {
	routes := make([]Route, 0, len(r.routes))
	for _, route := range r.routes {
		copied := *route
		copied.sourcePath = prefix + r.basePath + route.sourcePath
		routes = append(routes, &copied)
	}
	for _, subRouter := range r.subRouters {
		routes = append(routes, CollectRoutes(subRouter, prefix)...)
	}
	return routes
}
