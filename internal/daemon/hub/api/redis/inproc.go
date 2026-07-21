package redis

import (
	"context"
	"errors"
	"net"
	"sync"
)

const RedisInprocEndpoint = "redis+inproc://vine/hub"

type InprocDialer interface {
	DialInproc(ctx context.Context) (net.Conn, error)
}

var (
	inprocServer     InprocDialer
	inprocServerLock sync.RWMutex
)

func SetInprocServer(server InprocDialer) {
	inprocServerLock.Lock()
	defer inprocServerLock.Unlock()
	inprocServer = server
}

func InprocServer() InprocDialer {
	inprocServerLock.RLock()
	defer inprocServerLock.RUnlock()
	return inprocServer
}

func DialInproc(ctx context.Context, network string, addr string) (net.Conn, error) {
	server := InprocServer()
	if server == nil {
		return nil, errors.New("redis inproc server is not initialized")
	}
	return server.DialInproc(ctx)
}
