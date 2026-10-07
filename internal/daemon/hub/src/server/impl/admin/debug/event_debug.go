package debug

import (
	"strings"
	"uuid"

	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	skeled "go.yorun.ai/vine/internal/daemon/hub/api/skeled/admin"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

type EventDebugApiServiceServerImpl struct {
	skeled.DefaultEventDebugApiServiceServer

	CurrentApp     meta.CurrentApp        `inject:""`
	RegistryRepo   core.RegistryRepo      `inject:""`
	DescriptorRepo core.DescriptorRepo    `inject:""`
	NATSServer     *natsserver.NATSServer `inject:""`
	Flag           *hubflag.Flag          `inject:""`
}

func (s *EventDebugApiServiceServerImpl) defaultBuilder() _DebugDefaultBuilder {
	return _DebugDefaultBuilder{
		DescriptorRepo: s.DescriptorRepo,
	}
}

func (s *EventDebugApiServiceServerImpl) natsPublisher() _DebugNATSPublisher {
	return _DebugNATSPublisher{
		NATSServer: s.NATSServer,
		Flag:       s.Flag,
	}
}

func (s *EventDebugApiServiceServerImpl) ListEvents() []skeled.EventDebugEventItem {
	ret := []skeled.EventDebugEventItem{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, listener := range status.EventListeners {
			key := listener.EventSkelName + "\x00" + listener.DescriptorHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			eventDescriptor := s.findEventDescriptor(listener.EventSkelName, listener.DescriptorHash)
			ret = append(ret, toEventDebugEventItem(eventDescriptor, listener.DescriptorHash))
		}
	}
	return vslice.SortBy(ret, func(a skeled.EventDebugEventItem, b skeled.EventDebugEventItem) bool {
		if a.EventSkelName != b.EventSkelName {
			return strings.Compare(a.EventSkelName, b.EventSkelName) < 0
		}
		return strings.Compare(a.DescriptorHash, b.DescriptorHash) < 0
	})
}

func (s *EventDebugApiServiceServerImpl) BuildDefaultEmitRequest(eventSkelName string, descriptorHash string) skeled.EventDebugDefaultEmitRequest {
	eventDescriptor := s.findEventDescriptor(eventSkelName, descriptorHash)
	trace := meta.InitialTrace()
	return skeled.EventDebugDefaultEmitRequest{
		TraceId:   trace.Id(),
		SpanId:    trace.Span(),
		EventJson: s.defaultBuilder().defaultEventJson(eventDescriptor),
	}
}

func (s *EventDebugApiServiceServerImpl) EmitEvent(request skeled.EventDebugEmitRequest) {
	s.checkEventListener(request.EventSkelName, request.DescriptorHash)
	debugParseJson(string(request.EventJson))

	trace := debugTrace(request.TraceId, request.SpanId)
	msg := eventspec.NATSMessage{
		Metadata: eventspec.NATSMessageMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       s.CurrentApp.Name(),
			AppVersion:    s.CurrentApp.Version(),
			AppInstanceId: skeltype.NewUUID(uuid.MustParse(s.CurrentApp.InstanceId())),
			EmittedAt:     skeltype.NewTimestampNow(),
		},
		EventSkelName: request.EventSkelName,
		EventJson:     string(request.EventJson),
	}
	publisher := s.natsPublisher()
	publisher.publish(debugEventStreamConfig(), debugEventSubject(request.EventSkelName), vcode.MustMarshalJson(msg))
}

func (s *EventDebugApiServiceServerImpl) checkEventListener(eventSkelName string, descriptorHash string) {
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		if statusHasEventListener(status, eventSkelName, descriptorHash) {
			return
		}
	}
	ex.PanicNew(ex.NotFound, "event listener registration not found")
}

func (s *EventDebugApiServiceServerImpl) findEventDescriptor(eventSkelName string, descriptorHash string) *skeldesc.Event {
	for _, version := range s.DescriptorRepo.ListEventDescriptorVersions() {
		if version.Descriptor.SkelName == eventSkelName && (descriptorHash == "" || version.DescriptorHash == descriptorHash) {
			return version.Descriptor
		}
	}
	ex.PanicNew(ex.NotFound, "event descriptor not found")
	panic("unreachable")
}

func toEventDebugEventItem(event *skeldesc.Event, descriptorHash string) skeled.EventDebugEventItem {
	return skeled.EventDebugEventItem{
		Name:             event.Name,
		EventSkelName:    event.SkelName,
		DescriptorHash:   descriptorHash,
		Description:      event.Description,
		Deprecated:       event.Deprecated,
		DeprecatedReason: event.DeprecatedReason,
		Fields:           toDebugSkeletonFields(event.Members),
	}
}

func statusHasEventListener(status *core.AppStatus, eventSkelName string, descriptorHash string) bool {
	for _, listener := range status.EventListeners {
		if listener.EventSkelName == eventSkelName && (descriptorHash == "" || listener.DescriptorHash == descriptorHash) {
			return true
		}
	}
	return false
}
