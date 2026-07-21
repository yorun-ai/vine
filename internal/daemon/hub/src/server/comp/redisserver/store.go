package redisserver

import (
	"time"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
)

type _RedisStore interface {
	Start()
	Stop()

	Set(key string, value string)
	Get(key string) (string, bool)
	TTL(key string) int
	Scan(pattern string) []string
	Incr(key string) (int64, error)
	Del(key string) bool
	Expire(key string, seconds int) bool
	Publish(channel string, message string) int

	SetWithTTL(key string, value string, ttl time.Duration)
	InitRevision()
	SetAndNotify(key string, value string)
	SetWithTTLAndNotify(key string, value string, ttl time.Duration)
	DeleteAndNotify(key string)
	ApplyAndNotify(operations []hubredis.NotifyOperation)
	// Keep refreshes the TTL for an existing key. It returns false when the key is missing.
	Keep(key string, ttl time.Duration) bool

	ZAdd(key string, score float64, member string)
	// ZPopRangeByScore returns members in score order and removes exactly those members atomically.
	ZPopRangeByScore(key string, min float64, max float64, limit int) []string
	ZRem(key string, member string) bool
}
