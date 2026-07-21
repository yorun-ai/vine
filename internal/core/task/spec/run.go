package spec

type Run interface {
	Context() Context

	TriggerInfo() TriggerInfo
	TriggerImpl() TriggerImpl

	Arguments() any
	PositionalArguments() []any
}

type RunImpl struct {
	ContextValue Context

	TriggerInfoValue TriggerInfo
	TriggerImplValue TriggerImpl
	ArgumentsValue   any
}

func (r *RunImpl) Context() Context {
	return r.ContextValue
}

func (r *RunImpl) TriggerInfo() TriggerInfo {
	return r.TriggerInfoValue
}

func (r *RunImpl) TriggerImpl() TriggerImpl {
	return r.TriggerImplValue
}

func (r *RunImpl) Arguments() any {
	return r.ArgumentsValue
}

func (r *RunImpl) PositionalArguments() []any {
	return r.TriggerInfo().PositionArguments(r.ArgumentsValue)
}
