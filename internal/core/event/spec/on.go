package spec

type On interface {
	Context() Context

	EventInfo() EventInfo
	ListenerImpl() ListenerImpl
	EventPayload() any
}

type OnImpl struct {
	ContextValue Context

	EventInfoValue    EventInfo
	ListenerImplValue ListenerImpl
	EventPayloadValue any
}

func (o *OnImpl) Context() Context {
	return o.ContextValue
}

func (o *OnImpl) EventInfo() EventInfo {
	return o.EventInfoValue
}

func (o *OnImpl) ListenerImpl() ListenerImpl {
	return o.ListenerImplValue
}

func (o *OnImpl) EventPayload() any {
	return o.EventPayloadValue
}
