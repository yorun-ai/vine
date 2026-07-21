package spec

import "reflect"

type WebInfo interface {
	Name() string
	SkelName() string
	Hash() string
	ServerType() reflect.Type
	DefaultServerType() reflect.Type
}

type WebSpec struct {
	Name              string
	SkelName          string
	Hash              string
	ServerType        reflect.Type
	DefaultServerType reflect.Type

	info *_WebInfo
}

func (s *WebSpec) Info() WebInfo {
	return s.info
}

type _WebInfo struct {
	name              string
	skelName          string
	hash              string
	serverType        reflect.Type
	defaultServerType reflect.Type
}

func (i *_WebInfo) Name() string {
	return i.name
}

func (i *_WebInfo) SkelName() string {
	return i.skelName
}

func (i *_WebInfo) Hash() string {
	return i.hash
}

func (i *_WebInfo) ServerType() reflect.Type {
	return i.serverType
}

func (i *_WebInfo) DefaultServerType() reflect.Type {
	return i.defaultServerType
}
