package repo

import (
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func encodeFieldSources(sources core.FieldSources) string {
	if sources == nil {
		return "{}"
	}
	return vcode.MustMarshalJsonS(sources)
}
func decodeFieldSources(value string) core.FieldSources {
	if value == "" || value == "{}" {
		return nil
	}
	return vcode.MustUnmarshalJsonS[core.FieldSources](value)
}
