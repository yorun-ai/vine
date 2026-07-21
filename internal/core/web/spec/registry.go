package spec

import (
	"reflect"

	"go.yorun.ai/vine/internal/util/reflectutil"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

var webInfoBySkelName = map[string]WebInfo{}
var webInfoByDefaultEmbeddedType = map[reflect.Type]WebInfo{}

func Register(webSpec *WebSpec) {
	webInfo := initWebInfo(webSpec)
	vpre.CheckNil(webInfoBySkelName[webInfo.SkelName()], "web %s already registered", webInfo.SkelName())
	vpre.CheckNil(webInfoByDefaultEmbeddedType[webInfo.DefaultServerType().Elem()], "default web server type %s already registered", webInfo.DefaultServerType())

	webInfoBySkelName[webInfo.SkelName()] = webInfo
	webInfoByDefaultEmbeddedType[webInfo.DefaultServerType().Elem()] = webInfo
}

func RegisteredWebInfos() []WebInfo {
	return vmap.Values(webInfoBySkelName)
}

func GetWebInfo(handlerType reflect.Type) WebInfo {
	var webInfo WebInfo
	for _, embeddedType := range reflectutil.EmbeddedStructTypes(handlerType) {
		info := webInfoByDefaultEmbeddedType[embeddedType]
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
