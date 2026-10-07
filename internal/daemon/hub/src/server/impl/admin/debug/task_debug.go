package debug

import (
	"strings"
	"uuid"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	taskspec "go.yorun.ai/vine/internal/core/task/spec"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

type TaskDebugApiServiceServerImpl struct {
	skeled.DefaultTaskDebugApiServiceServer

	CurrentApp     meta.CurrentApp        `inject:""`
	RegistryRepo   core.RegistryRepo      `inject:""`
	DescriptorRepo core.DescriptorRepo    `inject:""`
	NATSServer     *natsserver.NATSServer `inject:""`
	Flag           *hubflag.Flag          `inject:""`
}

func (s *TaskDebugApiServiceServerImpl) defaultBuilder() _DebugDefaultBuilder {
	return _DebugDefaultBuilder{
		DescriptorRepo: s.DescriptorRepo,
	}
}

func (s *TaskDebugApiServiceServerImpl) natsPublisher() _DebugNATSPublisher {
	return _DebugNATSPublisher{
		NATSServer: s.NATSServer,
		Flag:       s.Flag,
	}
}

func (s *TaskDebugApiServiceServerImpl) ListTasks() []skeled.TaskDebugTaskItem {
	ret := []skeled.TaskDebugTaskItem{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, runner := range status.TaskRunners {
			key := runner.TaskSkelName + "\x00" + runner.DescriptorHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			taskDescriptor := s.findTaskDescriptor(runner.TaskSkelName, runner.DescriptorHash)
			ret = append(ret, skeled.TaskDebugTaskItem{
				Name:             taskDescriptor.Name,
				TaskSkelName:     taskDescriptor.SkelName,
				DescriptorHash:   runner.DescriptorHash,
				Description:      taskDescriptor.Description,
				Deprecated:       taskDescriptor.Deprecated,
				DeprecatedReason: taskDescriptor.DeprecatedReason,
			})
		}
	}
	return vslice.SortBy(ret, func(a skeled.TaskDebugTaskItem, b skeled.TaskDebugTaskItem) bool {
		if a.TaskSkelName != b.TaskSkelName {
			return strings.Compare(a.TaskSkelName, b.TaskSkelName) < 0
		}
		return strings.Compare(a.DescriptorHash, b.DescriptorHash) < 0
	})
}

func (s *TaskDebugApiServiceServerImpl) ListTriggers(taskSkelName string, descriptorHash string) []skeled.TaskDebugTriggerItem {
	taskDescriptor := s.findTaskDescriptor(taskSkelName, descriptorHash)
	ret := make([]skeled.TaskDebugTriggerItem, 0, len(taskDescriptor.Triggers))
	for _, trigger := range taskDescriptor.Triggers {
		ret = append(ret, toTaskDebugTriggerItem(trigger))
	}
	return vslice.SortBy(ret, func(a skeled.TaskDebugTriggerItem, b skeled.TaskDebugTriggerItem) bool {
		return strings.Compare(a.SkelName, b.SkelName) < 0
	})
}

func (s *TaskDebugApiServiceServerImpl) BuildDefaultLaunchRequest(taskSkelName string, descriptorHash string, triggerSkelName string) skeled.TaskDebugDefaultLaunchRequest {
	taskDescriptor := s.findTaskDescriptor(taskSkelName, descriptorHash)
	triggerDescriptor := s.findTriggerDescriptor(taskDescriptor, triggerSkelName)
	trace := meta.InitialTrace()
	return skeled.TaskDebugDefaultLaunchRequest{
		TraceId:       trace.Id(),
		SpanId:        trace.Span(),
		ArgumentsJson: s.defaultBuilder().defaultArgumentsJson(triggerDescriptor),
	}
}

func (s *TaskDebugApiServiceServerImpl) LaunchTask(request skeled.TaskDebugLaunchRequest) {
	s.checkTaskRunner(request.TaskSkelName, request.DescriptorHash)
	debugParseJson(string(request.ArgumentsJson))

	trace := debugTrace(request.TraceId, request.SpanId)
	msg := taskspec.NATSMessage{
		Metadata: taskspec.NATSMessageMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       s.CurrentApp.Name(),
			AppVersion:    s.CurrentApp.Version(),
			AppInstanceId: skeltype.NewUUID(uuid.MustParse(s.CurrentApp.InstanceId())),
			LaunchedAt:    skeltype.NewTimestampNow(),
		},
		TaskSkelName:    request.TaskSkelName,
		TriggerSkelName: request.TriggerSkelName,
		ArgumentsJson:   string(request.ArgumentsJson),
	}
	publisher := s.natsPublisher()
	publisher.publish(debugTaskStreamConfig(), debugTaskSubject(request.TaskSkelName), vcode.MustMarshalJson(msg))
}

func (s *TaskDebugApiServiceServerImpl) checkTaskRunner(taskSkelName string, descriptorHash string) {
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		if statusHasTaskRunner(status, taskSkelName, descriptorHash) {
			return
		}
	}
	ex.PanicNew(ex.NotFound, "task runner registration not found")
}

func (s *TaskDebugApiServiceServerImpl) findTaskDescriptor(taskSkelName string, descriptorHash string) *skeldesc.Task {
	for _, version := range s.DescriptorRepo.ListTaskDescriptorVersions() {
		if version.Descriptor.SkelName == taskSkelName && (descriptorHash == "" || version.DescriptorHash == descriptorHash) {
			return version.Descriptor
		}
	}
	ex.PanicNew(ex.NotFound, "task descriptor not found")
	panic("unreachable")
}

func (s *TaskDebugApiServiceServerImpl) findTriggerDescriptor(taskDescriptor *skeldesc.Task, triggerSkelName string) *skeldesc.TaskTrigger {
	for _, trigger := range taskDescriptor.Triggers {
		if trigger.SkelName == triggerSkelName {
			return trigger
		}
	}
	ex.PanicNew(ex.NotFound, "trigger descriptor not found")
	panic("unreachable")
}

func toTaskDebugTriggerItem(trigger *skeldesc.TaskTrigger) skeled.TaskDebugTriggerItem {
	return skeled.TaskDebugTriggerItem{
		Name:             trigger.Name,
		SkelName:         trigger.SkelName,
		Description:      trigger.Description,
		Deprecated:       trigger.Deprecated,
		DeprecatedReason: trigger.DeprecatedReason,
		InputDescription: trigger.InputDescription,
		Arguments:        toDebugSkeletonFields(trigger.Arguments),
	}
}

func statusHasTaskRunner(status *core.AppStatus, taskSkelName string, descriptorHash string) bool {
	for _, runner := range status.TaskRunners {
		if runner.TaskSkelName == taskSkelName && (descriptorHash == "" || runner.DescriptorHash == descriptorHash) {
			return true
		}
	}
	return false
}
