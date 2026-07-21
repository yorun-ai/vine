package redisserver

import (
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/util/vpre"
)

type NotifyBatch struct {
	server     *Server
	operations []hubredis.NotifyOperation
	notified   bool
}

func (s *Server) NotifyBatch() *NotifyBatch {
	return &NotifyBatch{server: s}
}

func (b *NotifyBatch) Set(key string, value string) {
	b.checkNotified()
	b.operations = append(b.operations, hubredis.NotifyOperation{
		Key:   key,
		Value: value,
	})
}

func (b *NotifyBatch) Delete(key string) {
	b.checkNotified()
	b.operations = append(b.operations, hubredis.NotifyOperation{
		Key:    key,
		Delete: true,
	})
}

func (b *NotifyBatch) Notify() {
	b.checkNotified()
	b.notified = true
	b.server.applyAndNotify(b.operations)
}

func (b *NotifyBatch) checkNotified() {
	vpre.Check(!b.notified, "redis notify batch already notified")
}
