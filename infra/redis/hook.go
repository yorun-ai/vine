package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

var errSelectDatabase = errors.New("redis SELECT cannot change the configured database; use another endpoint")

// _SelectHook keeps pooled connections on their configured database. go-redis
// also invokes hooks for its initialization SELECT, so selecting the configured
// database remains allowed. This applies to both external and memory clients.
type _SelectHook struct {
	dbIndex int
}

func (h _SelectHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h _SelectHook) rejects(cmd goredis.Cmder) bool {
	if !strings.EqualFold(cmd.Name(), "select") {
		return false
	}
	args := cmd.Args()
	if len(args) != 2 {
		return true
	}
	arg := args[1]
	if bytes, ok := arg.([]byte); ok {
		arg = string(bytes)
	}
	index, err := strconv.Atoi(fmt.Sprint(arg))
	return err != nil || index != h.dbIndex
}

func (h _SelectHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		if h.rejects(cmd) {
			cmd.SetErr(errSelectDatabase)
			return errSelectDatabase
		}
		return next(ctx, cmd)
	}
}

func (h _SelectHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		for _, cmd := range cmds {
			if h.rejects(cmd) {
				// No command is sent, so all queued results must report the rejection.
				for _, queued := range cmds {
					queued.SetErr(errSelectDatabase)
				}
				return errSelectDatabase
			}
		}
		return next(ctx, cmds)
	}
}
