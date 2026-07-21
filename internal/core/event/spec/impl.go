package spec

import (
	"fmt"
	"reflect"

	"go.yorun.ai/vine/util/vpre"
)

type ListenerImplDict struct {
	listenerByName map[string]*_ListenerImpl
	listenerByInfo map[EventInfo]*_ListenerImpl
}

func NewListenerImplDict() *ListenerImplDict {
	return &ListenerImplDict{
		listenerByName: map[string]*_ListenerImpl{},
		listenerByInfo: map[EventInfo]*_ListenerImpl{},
	}
}

func (d *ListenerImplDict) Add(listenerImplType reflect.Type) {
	checkImplType(listenerImplType)
	eventInfo, isERType := getEventInfo(listenerImplType)
	_, ok := d.listenerByName[eventInfo.SkelName()]
	vpre.Check(!ok, "event listener %s already added", eventInfo.SkelName())

	method, ok := listenerImplType.MethodByName(eventInfo.ListenerMethodName())
	vpre.Check(ok, "event listener method %s not found on %s", eventInfo.ListenerMethodName(), listenerImplType.String())
	listenerImpl := &_ListenerImpl{
		kind:     listenerImplType,
		method:   method,
		isERType: isERType,
		info:     eventInfo,
	}
	d.listenerByName[eventInfo.SkelName()] = listenerImpl
	d.listenerByInfo[eventInfo] = listenerImpl
}

func checkImplType(implType reflect.Type) {
	vpre.Check(implType.Kind() == reflect.Pointer && implType.Elem().Kind() == reflect.Struct, "event listener impl type %s must be a pointer to struct", implType)
}

func (d *ListenerImplDict) GetListenerImpl(eventSkelName string) (ListenerImpl, error) {
	listenerImpl, ok := d.listenerByName[eventSkelName]
	if !ok {
		return nil, fmt.Errorf("event %s not found", eventSkelName)
	}
	return listenerImpl, nil
}

func (d *ListenerImplDict) GetListenerImplByInfo(eventInfo EventInfo) (ListenerImpl, error) {
	if eventInfo == nil {
		return nil, fmt.Errorf("event info is nil")
	}
	listenerImpl, ok := d.listenerByInfo[eventInfo]
	if !ok {
		return nil, fmt.Errorf("event %s not found", eventInfo.SkelName())
	}
	return listenerImpl, nil
}

func (d *ListenerImplDict) IterateListenerImpl(iterate func(info ListenerImpl)) {
	for _, info := range d.listenerByName {
		iterate(info)
	}
}

type ListenerImpl interface {
	Type() reflect.Type
	Method() reflect.Method
	IsERType() bool
	Info() EventInfo
}

type _ListenerImpl struct {
	kind     reflect.Type
	method   reflect.Method
	isERType bool
	info     EventInfo
}

func (i *_ListenerImpl) Type() reflect.Type {
	return i.kind
}

func (i *_ListenerImpl) Method() reflect.Method {
	return i.method
}

func (i *_ListenerImpl) IsERType() bool {
	return i.isERType
}

func (i *_ListenerImpl) Info() EventInfo {
	return i.info
}
