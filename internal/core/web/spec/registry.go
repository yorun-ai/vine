package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

type Registry struct {
	webInfoBySkelName            map[string]WebInfo
	webInfoByDefaultEmbeddedType map[reflect.Type]WebInfo
}

func NewRegistry() *Registry {
	return &Registry{
		webInfoBySkelName:            map[string]WebInfo{},
		webInfoByDefaultEmbeddedType: map[reflect.Type]WebInfo{},
	}
}

var defaultRegistry = NewRegistry()

func Register(webSpec *WebSpec) {
	defaultRegistry.Register(webSpec)
}

func (r *Registry) Register(webSpec *WebSpec) {
	webInfo := initWebInfo(webSpec)
	vpre.CheckNil(r.webInfoBySkelName[webInfo.SkelName()], "web %s already registered", webInfo.SkelName())
	vpre.CheckNil(r.webInfoByDefaultEmbeddedType[webInfo.DefaultServerType().Elem()], "default web server type %s already registered", webInfo.DefaultServerType())

	r.webInfoBySkelName[webInfo.SkelName()] = webInfo
	r.webInfoByDefaultEmbeddedType[webInfo.DefaultServerType().Elem()] = webInfo
}

func RegisteredWebInfos() []WebInfo {
	return defaultRegistry.RegisteredWebInfos()
}

func (r *Registry) RegisteredWebInfos() []WebInfo {
	return vmap.Values(r.webInfoBySkelName)
}

func GetWebInfo(handlerType reflect.Type) WebInfo {
	return defaultRegistry.GetWebInfo(handlerType)
}

func (r *Registry) GetWebInfo(handlerType reflect.Type) WebInfo {
	var webInfo WebInfo
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(handlerType) {
		info := r.webInfoByDefaultEmbeddedType[embeddedType]
		if info == nil {
			continue
		}
		vpre.CheckNil(webInfo, "multiple embedded default web server type found on %s", handlerType)
		webInfo = info
	}
	vpre.CheckNotNil(webInfo, "no embedded default web server type found on %s", handlerType)
	return webInfo
}

func initWebInfo(webSpec *WebSpec) *_WebInfo {
	if webSpec == nil {
		return nil
	}
	info := &_WebInfo{
		name:              webSpec.Name,
		skelName:          webSpec.SkelName,
		hash:              webSpec.Hash,
		serverType:        webSpec.ServerType,
		defaultServerType: webSpec.DefaultServerType,
	}
	webSpec.info = info
	return info
}
