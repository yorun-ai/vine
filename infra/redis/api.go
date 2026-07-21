package redis

import (
	"context"

	internalredis "go.yorun.ai/vine/internal/infra/redis"
)

// Option configures a Redis component.
type Option = internalredis.Option

// TypeAdder adds a Redis capability type to a specification.
type TypeAdder = internalredis.TypeAdder

// RedisSpec describes a named Redis connection and its capabilities.
type RedisSpec = internalredis.RedisSpec

// Redis wraps a Redis client for Vine-managed execution contexts.
type Redis = internalredis.Redis

// Cache stores typed values under a shared key prefix.
type Cache[T any] = internalredis.Cache[T]

// Locker creates distributed Redis locks.
type Locker = internalredis.Locker

// Lock represents one distributed lock acquisition.
type Lock = internalredis.Lock

// NewCache creates a typed cache using redis, ctx, and keyPrefix.
func NewCache[T any](redis *Redis, ctx context.Context, keyPrefix string) *Cache[T] {
	return internalredis.NewCache[T](redis, ctx, keyPrefix)
}
