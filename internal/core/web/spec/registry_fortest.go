package spec

import "reflect"

func ResetRegistryForTest() {
	webInfoBySkelName = map[string]WebInfo{}
	webInfoByDefaultEmbeddedType = map[reflect.Type]WebInfo{}
}
