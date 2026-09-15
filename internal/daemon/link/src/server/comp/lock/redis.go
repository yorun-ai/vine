package lock

import (
	"context"
	"errors"
	"net"

	goredis "github.com/redis/go-redis/v9"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/meta"
	"go.yorun.ai/vine/util/vpre"
)

type _RedisLocker struct {
	client *goredis.Client
}

const acquireScript = `
local value = redis.call("get", KEYS[1])
if value then
 return 0
end
redis.call("psetex", KEYS[1], ARGV[2], ARGV[1])
return 1
`

const renewScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
 return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0
`

const releaseScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
 return redis.call("del", KEYS[1])
end
return 0
`

func newRedisLocker(endpoint string) *_RedisLocker {
	options, err := goredis.ParseURL(endpoint)
	vpre.Check(err == nil, "invalid lock Redis endpoint from Hub")
	// Respect request deadlines and avoid transparent retries extending a renewed lease.
	options.ContextTimeoutEnabled = true
	options.MaxRetries = -1
	return new(_RedisLocker{client: goredis.NewClient(options)})
}

func (l *_RedisLocker) Acquire(ctx meta.Context, key, token string, ttlMillis int) bool {
	return l.eval(ctx, acquireScript, key, token, ttlMillis)
}

func (l *_RedisLocker) Renew(ctx meta.Context, key, token string, ttlMillis int) bool {
	return l.eval(ctx, renewScript, key, token, ttlMillis)
}

func (l *_RedisLocker) Release(ctx meta.Context, key, token string) bool {
	return l.eval(ctx, releaseScript, key, token)
}

func (l *_RedisLocker) eval(ctx meta.Context, script, key string, args ...any) bool {
	result, err := l.client.Eval(ctx, script, []string{"vine:lock:" + key}, args...).Int64()
	if err != nil {
		code := ex.ServiceUnavailable
		message := "Redis lock service is unavailable"
		var netErr net.Error
		switch {
		case errors.Is(err, context.Canceled):
			code = ex.InvocationCancelled
			message = "Redis lock operation was cancelled"
		case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
			code = ex.InvocationTimeout
			message = "Redis lock operation timed out"
		}
		ex.PanicNew(code, message, ex.WithCause(err))
	}
	return result == 1
}

func (l *_RedisLocker) Close() {
	_ = l.client.Close()
}
