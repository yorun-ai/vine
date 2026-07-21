package log

import "sync/atomic"

var inprocClientLogEnabled atomic.Bool

func init() {
	DisableInprocClientLog()
}

func EnableInprocClientLog() {
	inprocClientLogEnabled.Store(true)
}

func DisableInprocClientLog() {
	inprocClientLogEnabled.Store(false)
}

func IsInprocClientLogEnabled() bool {
	return inprocClientLogEnabled.Load()
}
