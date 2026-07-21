package event

import (
	"github.com/google/uuid"

	eventlog "go.yorun.ai/vine/internal/core/event/log"
	"go.yorun.ai/vine/internal/core/event/spec"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

type EmitterOption struct {
	Context     meta.Context
	ClientApp   meta.App
	Logger      *logger.Logger
	EventClient linkskeled.EventServiceClient
}

type EmitOption interface {
	apply(options *_EmitOptions)
}

type _EmitOptionFunc func(options *_EmitOptions)

func (f _EmitOptionFunc) apply(options *_EmitOptions) {
	f(options)
}

type _EmitOptions struct{}

func newEmitOptions() *_EmitOptions { return &_EmitOptions{} }

type Emitter struct {
	context     meta.Context
	clientApp   meta.App
	logger      *logger.Logger
	eventClient linkskeled.EventServiceClient
}

func NewEmitter(option EmitterOption) *Emitter {
	vpre.CheckNotNil(option.Context, "event emitter context cannot be nil")
	vpre.CheckNotNil(option.Logger, "event emitter logger cannot be nil")
	vpre.CheckNotNil(option.EventClient, "event emitter client cannot be nil")
	return &Emitter{
		context:     option.Context,
		clientApp:   option.ClientApp,
		logger:      option.Logger,
		eventClient: option.EventClient,
	}
}

func (e *Emitter) Emit(eventInfo spec.EventInfo, eventPayload any, options ...EmitOption) {
	emission := e.buildEmission(eventInfo, eventPayload, options)
	e.eventClient.EmitEvent(emission, rpcclient.WithContext(e.context))
	eventlog.EmitterEmitSuccess(e.logger, e.context.Trace(), eventInfo, e.clientApp)
}

func (e *Emitter) buildEmission(eventInfo spec.EventInfo, eventPayload any, options []EmitOption) linkskeled.EventEmission {
	emitOptions := newEmitOptions()
	for _, option := range options {
		option.apply(emitOptions)
	}

	return linkskeled.EventEmission{
		Metadata: linkskeled.EventEmissionMeta{
			TraceId:       e.context.Trace().Id(),
			TraceSpan:     e.context.Trace().Span(),
			AppName:       e.clientApp.Name(),
			AppVersion:    e.clientApp.Version(),
			AppInstanceId: skel.NewUUID(uuid.MustParse(e.clientApp.InstanceId())),
		},
		EventSkelName: eventInfo.SkelName(),
		EventJson:     vcode.MustMarshalJsonS(eventPayload),
	}
}
