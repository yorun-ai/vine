package debug

import (
	"strings"

	"github.com/google/uuid"
	eventspec "go.yorun.ai/vine/internal/core/event/spec"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/natsserver"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vslice"
)

type EventDebugServiceServerImpl struct {
	skeled.DefaultEventDebugServiceServer

	RegistryRepo core.RegistryRepo      `inject:""`
	SchemaRepo   core.SchemaRepo        `inject:""`
	NATSServer   *natsserver.NATSServer `inject:""`
	Flag         *hubflag.Flag          `inject:""`
}

func (s *EventDebugServiceServerImpl) defaultBuilder() _DebugDefaultBuilder {
	return _DebugDefaultBuilder{SchemaRepo: s.SchemaRepo}
}

func (s *EventDebugServiceServerImpl) natsPublisher() _DebugNATSPublisher {
	return _DebugNATSPublisher{
		NATSServer: s.NATSServer,
		Flag:       s.Flag,
	}
}

func (s *EventDebugServiceServerImpl) ListEvents() []skeled.EventDebugEventItem {
	ret := []skeled.EventDebugEventItem{}
	seen := map[string]struct{}{}
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		for _, listener := range status.EventListeners {
			key := listener.EventSkelName + "\x00" + listener.SchemaHash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			eventSchema := s.findEventSchema(listener.EventSkelName, listener.SchemaHash)
			ret = append(ret, toEventDebugEventItem(eventSchema, listener.SchemaHash))
		}
	}
	return vslice.SortBy(ret, func(a skeled.EventDebugEventItem, b skeled.EventDebugEventItem) bool {
		if a.EventSkelName != b.EventSkelName {
			return strings.Compare(a.EventSkelName, b.EventSkelName) < 0
		}
		return strings.Compare(a.SchemaHash, b.SchemaHash) < 0
	})
}

func (s *EventDebugServiceServerImpl) BuildDefaultEmitRequest(eventSkelName string, schemaHash string) skeled.EventDebugDefaultEmitRequest {
	eventSchema := s.findEventSchema(eventSkelName, schemaHash)
	trace := meta.InitialTrace()
	return skeled.EventDebugDefaultEmitRequest{
		TraceId:   trace.Id(),
		SpanId:    trace.Span(),
		EventJson: s.defaultBuilder().defaultEventJson(eventSchema),
	}
}

func (s *EventDebugServiceServerImpl) EmitEvent(request skeled.EventDebugEmitRequest) {
	s.checkEventListener(request.EventSkelName, request.SchemaHash)
	debugParseJson(string(request.EventJson))

	trace := debugTrace(request.TraceId, request.SpanId)
	msg := eventspec.NATSMessage{
		Metadata: eventspec.NATSMessageMeta{
			TraceId:       trace.Id(),
			TraceSpan:     trace.Span(),
			AppName:       debugClientName,
			AppVersion:    debugClientVersion,
			AppInstanceId: skel.NewUUID(uuid.MustParse(debugClientInstanceId)),
			EmittedAt:     skel.NewTimestampNow(),
		},
		EventSkelName: request.EventSkelName,
		EventJson:     string(request.EventJson),
	}
	publisher := s.natsPublisher()
	publisher.publish(debugEventStreamConfig(), debugEventSubject(request.EventSkelName), vcode.MustMarshalJson(msg))
}

func (s *EventDebugServiceServerImpl) checkEventListener(eventSkelName string, schemaHash string) {
	for _, status := range s.RegistryRepo.ListAppStatuses() {
		if statusHasEventListener(status, eventSkelName, schemaHash) {
			return
		}
	}
	ex.PanicNew(ex.NotFound, "event listener registration not found")
}

func (s *EventDebugServiceServerImpl) findEventSchema(eventSkelName string, schemaHash string) *skel.EventSchema {
	for _, version := range s.SchemaRepo.ListEventSchemaVersions() {
		if version.Schema.SkelName == eventSkelName && (schemaHash == "" || version.SchemaHash == schemaHash) {
			return version.Schema
		}
	}
	ex.PanicNew(ex.NotFound, "event schema not found")
	panic("unreachable")
}

func toEventDebugEventItem(event *skel.EventSchema, schemaHash string) skeled.EventDebugEventItem {
	return skeled.EventDebugEventItem{
		Name:          event.Name,
		EventSkelName: event.SkelName,
		SchemaHash:    schemaHash,
		Description:   optionalString(event.Description),
		Fields:        toDebugSkeletonFields(event.Members),
	}
}

func statusHasEventListener(status *core.AppStatus, eventSkelName string, schemaHash string) bool {
	for _, listener := range status.EventListeners {
		if listener.EventSkelName == eventSkelName && (schemaHash == "" || listener.SchemaHash == schemaHash) {
			return true
		}
	}
	return false
}
