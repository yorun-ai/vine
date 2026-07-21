package server

import (
	"context"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/ctr"
	"go.yorun.ai/vine/internal/core/di"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/web/spec"
)

type Executor interface {
	Init(routeDict []spec.RouteInfo)
	Execute(route spec.RouteInfo, ginCtx *gin.Context)
}

type _ContainerExecutor struct {
	filterTypes  []reflect.Type
	bindAppliers []di.BindApplier
	container    ctr.Container
}

func NewContainerExecutor(filterTypes []reflect.Type, bindAppliers []di.BindApplier) Executor {
	return &_ContainerExecutor{
		filterTypes:  filterTypes,
		bindAppliers: bindAppliers,
	}
}

func (e *_ContainerExecutor) Init(routeDict []spec.RouteInfo) {
	handlerTypeSet := map[reflect.Type]struct{}{}
	for _, route := range routeDict {
		handlerTypeSet[route.HandlerType()] = struct{}{}
	}

	bindAppliers := []di.BindApplier{
		func(b *di.Binder) {
			b.Bind(di.T[spec.Context]()).In(di.ExecutionScope)
			b.BindFactory(func(ctx spec.Context) *gin.Context { return ctx.Gin() }).In(di.ExecutionScope)
			for handlerType := range handlerTypeSet {
				b.Bind(handlerType)
			}
		},
	}
	bindAppliers = append(bindAppliers, e.bindAppliers...)

	e.container = ctr.NewContainer(ctr.Option{
		BindAppliers: bindAppliers,
		FilterTypes:  e.filterTypes,
	})
}

func (e *_ContainerExecutor) Execute(route spec.RouteInfo, ginCtx *gin.Context) {
	cancel, err := applyRequestOptions(ginCtx)
	if err != nil {
		logger.Warn(err.Error())
		ginCtx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	defer cancel()

	requestMeta, err := spec.DecodeRequestMeta(ginCtx.Request.Header)
	if err != nil {
		logger.Warn(err.Error())
		ginCtx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	requestMeta.Trace = requestMeta.Trace.NewChildTrace()

	webCtx := spec.NewContext(ginCtx, route, requestMeta.Trace, requestMeta.Initiator, requestMeta.Actor)
	spec.EncodeTraceToHeader(ginCtx.Writer.Header(), requestMeta.Trace)

	exe := e.container.NewExecution(route.HandlerType(), route.HandlerMethod())
	exe.Execute(nil, func(s *di.Seeder) {
		s.Seed(di.T[spec.Context](), webCtx)
	})
}

func applyRequestOptions(ginCtx *gin.Context) (context.CancelFunc, error) {
	options, err := spec.DecodeOptionsFromHeader(ginCtx.Request.Header)
	if err != nil {
		return func() {}, err
	}
	if options.Timeout <= 0 {
		return func() {}, nil
	}

	reqContext, cancel := context.WithTimeout(ginCtx.Request.Context(), options.Timeout)
	ginCtx.Request = ginCtx.Request.WithContext(reqContext)
	return cancel, nil
}
