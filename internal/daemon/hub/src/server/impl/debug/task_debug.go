package debug

import (
	"strings"

	"github.com/google/uuid"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

type TaskDebugServiceServerImpl struct {
	skeled.DefaultTaskDebugServiceServer

	RegistryRepo core.RegistryRepo      `inject:""`
	SchemaRepo   core.SchemaRepo        `inject:""`
	NATSServer   *natsserver.NATSServer `inject:""`
	Flag         *hubflag.Flag          `inject:""`
}

func (s *TaskDebugServiceServerImpl) defaultBuilder() _DebugDefaultBuilder {
	return _DebugDefaultBuilder{SchemaRepo: s.SchemaRepo}
}

func (s *TaskDebugServiceServerImpl) natsPublisher() _DebugNATSPublisher {
	return _DebugNATSPublisher{
		NATSServer: s.NATSServer,
		Flag:       s.Flag,
	}
}

func (s *TaskDebugServiceServerImpl) ListTasks() []skeled.TaskDebugTaskItem {
	ret := []skeled.TaskDebugTaskItem{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, runner := range status.TaskRunners {
			key := runner.TaskSkelName + "\x00" + runner.SchemaHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			taskSchema := s.findTaskSchema(runner.TaskSkelName, runner.SchemaHash)
			ret = append(ret, skeled.TaskDebugTaskItem{
				Name:         taskSchema.Name,
				TaskSkelName: taskSchema.SkelName,
				SchemaHash:   runner.SchemaHash,
				Description:  optionalString(taskSchema.Description),
			})
		}
	}
	return vslice.SortBy(ret, func(a skeled.TaskDebugTaskItem, b skeled.TaskDebugTaskItem) bool {
		if a.TaskSkelName != b.TaskSkelName {
			return strings.Compare(a.TaskSkelName, b.TaskSkelName) < 0
		}
		return strings.Compare(a.SchemaHash, b.SchemaHash) < 0
	})
}

func (s *TaskDebugServiceServerImpl) ListTriggers(taskSkelName string, schemaHash string) []skeled.TaskDebugTriggerItem {
	taskSchema := s.findTaskSchema(taskSkelName, schemaHash)
	ret := make([]skeled.TaskDebugTriggerItem, 0, len(taskSchema.Triggers))
	for _, trigger := range taskSchema.Triggers {
		ret = append(ret, toTaskDebugTriggerItem(trigger))
	}
	return vslice.SortBy(ret, func(a skeled.TaskDebugTriggerItem, b skeled.TaskDebugTriggerItem) bool {
		return strings.Compare(a.SkelName, b.SkelName) < 0
	})
}

func (s *TaskDebugServiceServerImpl) BuildDefaultLaunchRequest(taskSkelName string, schemaHash string, triggerSkelName string) skeled.TaskDebugDefaultLaunchRequest {
	taskSchema := s.findTaskSchema(taskSkelName, schemaHash)
	triggerSchema := s.findTriggerSchema(taskSchema, triggerSkelName)
	trace := meta.InitialTrace()
	return skeled.TaskDebugDefaultLaunchRequest{
		TraceId:       trace.Id(),
		SpanId:        trace.Span(),
		ArgumentsJson: s.defaultBuilder().defaultArgumentsJson(triggerSchema),
	}
}

func (s *TaskDebugServiceServerImpl) LaunchTask(request skeled.TaskDebugLaunchRequest) {
	s.checkTaskRunner(request.TaskSkelName, request.SchemaHash)
	debugParseJson(string(request.ArgumentsJson))

	trace := debugTrace(request.TraceId, request.SpanId)
	msg := taskspec.NATSMessage{
		Metadata: taskspec.NATSMessageMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       debugClientName,
			AppVersion:    debugClientVersion,
			AppInstanceId: skel.NewUUID(uuid.MustParse(debugClientInstanceId)),
			LaunchedAt:    skel.NewTimestampNow(),
		},
		TaskSkelName:    request.TaskSkelName,
		TriggerSkelName: request.TriggerSkelName,
		ArgumentsJson:   string(request.ArgumentsJson),
	}
	publisher := s.natsPublisher()
	publisher.publish(debugTaskStreamConfig(), debugTaskSubject(request.TaskSkelName), vcode.MustMarshalJson(msg))
}

func (s *TaskDebugServiceServerImpl) checkTaskRunner(taskSkelName string, schemaHash string) {
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		if statusHasTaskRunner(status, taskSkelName, schemaHash) {
			return
		}
	}
	ex.PanicNew(ex.NotFound, "task runner registration not found")
}

func (s *TaskDebugServiceServerImpl) findTaskSchema(taskSkelName string, schemaHash string) *skel.TaskSchema {
	for _, version := range s.SchemaRepo.ListTaskSchemaVersions() {
		if version.Schema.SkelName == taskSkelName && (schemaHash == "" || version.SchemaHash == schemaHash) {
			return version.Schema
		}
	}
	ex.PanicNew(ex.NotFound, "task schema not found")
	panic("unreachable")
}

func (s *TaskDebugServiceServerImpl) findTriggerSchema(taskSchema *skel.TaskSchema, triggerSkelName string) *skel.TriggerSchema {
	for _, trigger := range taskSchema.Triggers {
		if trigger.SkelName == triggerSkelName {
			return trigger
		}
	}
	ex.PanicNew(ex.NotFound, "trigger schema not found")
	panic("unreachable")
}

func toTaskDebugTriggerItem(trigger *skel.TriggerSchema) skeled.TaskDebugTriggerItem {
	return skeled.TaskDebugTriggerItem{
		Name:             trigger.Name,
		SkelName:         trigger.SkelName,
		Description:      optionalString(trigger.Description),
		InputDescription: optionalString(trigger.InputDescription),
		Example:          optionalString(trigger.Example),
		Arguments:        toDebugSkeletonFields(trigger.Arguments),
	}
}

func statusHasTaskRunner(status *core.AppStatus, taskSkelName string, schemaHash string) bool {
	for _, runner := range status.TaskRunners {
		if runner.TaskSkelName == taskSkelName && (schemaHash == "" || runner.SchemaHash == schemaHash) {
			return true
		}
	}
	return false
}
