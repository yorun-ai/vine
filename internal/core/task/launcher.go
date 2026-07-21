package task

import (
	"github.com/google/uuid"
	linkskeled "go.yorun.ai/vine/internal/core/link/skeled"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/meta"
	rpcclient "go.yorun.ai/vine/internal/core/rpc/client"
	"go.yorun.ai/vine/internal/core/skel"
	tasklog "go.yorun.ai/vine/internal/core/task/log"
	"go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vpre"
)

// Option

type LauncherOption struct {
	Context    meta.Context
	ClientApp  meta.App
	Logger     *logger.Logger
	TaskClient linkskeled.TaskServiceClient
}

type LaunchOption interface {
	apply(options *_LaunchOptions)
}

type _LaunchOptionFunc func(options *_LaunchOptions)

func (f _LaunchOptionFunc) apply(options *_LaunchOptions) {
	f(options)
}

type _LaunchOptions struct{}

func newLaunchOptions() *_LaunchOptions { return &_LaunchOptions{} }

// Launcher

type Launcher struct {
	context    meta.Context
	clientApp  meta.App
	logger     *logger.Logger
	taskClient linkskeled.TaskServiceClient
}

func NewLauncher(option LauncherOption) *Launcher {
	vpre.CheckNotNil(option.Context, "task launcher context cannot be nil")
	vpre.CheckNotNil(option.Logger, "task launcher logger cannot be nil")
	vpre.CheckNotNil(option.TaskClient, "task launcher task client cannot be nil")
	return &Launcher{
		context:    option.Context,
		clientApp:  option.ClientApp,
		logger:     option.Logger,
		taskClient: option.TaskClient,
	}
}

func (l *Launcher) Launch(triggerInfo spec.TriggerInfo, arguments any, options ...LaunchOption) {
	launch := l.buildLaunch(triggerInfo, arguments, options)
	l.taskClient.LaunchTask(launch, rpcclient.WithContext(l.context))
	tasklog.LauncherLaunchSuccess(l.logger, l.context.Trace(), triggerInfo, l.clientApp)
}

func (l *Launcher) buildLaunch(triggerInfo spec.TriggerInfo, arguments any, options []LaunchOption) linkskeled.TaskLaunch {
	launchOptions := newLaunchOptions()
	for _, option := range options {
		option.apply(launchOptions)
	}

	if arguments == nil {
		arguments = triggerInfo.NewArguments()
	}
	vpre.CheckNilError(triggerInfo.ValidateArguments(arguments), "arguments validation failed")

	return linkskeled.TaskLaunch{
		Metadata: linkskeled.TaskLaunchMeta{
			TraceId:       l.context.Trace().Id(),
			TraceSpan:     l.context.Trace().Span(),
			AppName:       l.clientApp.Name(),
			AppVersion:    l.clientApp.Version(),
			AppInstanceId: skel.NewUUID(uuid.MustParse(l.clientApp.InstanceId())),
		},
		TaskSkelName:    triggerInfo.Task().SkelName(),
		TriggerSkelName: triggerInfo.SkelName(),
		ArgumentsJson:   vcode.MustMarshalJsonS(arguments),
	}
}
