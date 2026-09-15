package lockserver

import (
	"sync"
	"time"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/ex"
	hublock "go.yorun.ai/vine/internal/daemon/hub/api/lock"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

// Server owns ephemeral leases exposed by the Hub Control API.
type Server struct {
	app.BaseComponent
	Flag *flag.Flag `inject:""`

	mutex sync.Mutex
	items map[string]_Lease
	stop  chan struct{}
	done  chan struct{}
	once  sync.Once
}

type _Lease struct {
	token   string
	expires time.Time
}

func (s *Server) DIInit() {
	if s.Flag.LockMode != hublock.ModeEmbedded {
		return
	}
	s.items = make(map[string]_Lease)
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.sweep()
}

func (s *Server) checkRequest(key string, token string) {
	ex.PanicNewIfNot(s.Flag.LockMode == hublock.ModeEmbedded, ex.ServiceUnavailable, "Hub embedded lock service is not enabled")
	ex.PanicNewIfNot(key != "" && token != "", ex.InvalidRequest, "lock key and token must not be empty")
}

func leaseDuration(ttlMillis int) time.Duration {
	ex.PanicNewIfNot(ttlMillis > 0 && int64(ttlMillis) <= int64((1<<63-1)/time.Millisecond), ex.InvalidRequest, "invalid lock lease duration")
	return time.Duration(ttlMillis) * time.Millisecond
}

func (s *Server) Acquire(key string, token string, ttlMillis int) bool {
	s.checkRequest(key, token)
	ttl := leaseDuration(ttlMillis)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	if lease, exists := s.items[key]; exists && now.Before(lease.expires) {
		// An active lease blocks acquisition even for the same token; use Renew to extend it.
		return false
	}
	s.items[key] = _Lease{token: token, expires: now.Add(ttl)}
	return true
}

func (s *Server) Renew(key string, token string, ttlMillis int) bool {
	s.checkRequest(key, token)
	ttl := leaseDuration(ttlMillis)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	now := time.Now()
	lease, exists := s.items[key]
	if !exists || !now.Before(lease.expires) || lease.token != token {
		return false
	}
	lease.expires = now.Add(ttl)
	s.items[key] = lease
	return true
}

func (s *Server) Release(key string, token string) bool {
	s.checkRequest(key, token)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	lease, exists := s.items[key]
	if !exists || !time.Now().Before(lease.expires) || lease.token != token {
		return false
	}
	delete(s.items, key)
	return true
}

func (s *Server) sweep() {
	defer close(s.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			s.mutex.Lock()
			for key, lease := range s.items {
				if !now.Before(lease.expires) {
					delete(s.items, key)
				}
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Server) AfterAppStop() {
	if s.stop == nil {
		return
	}
	s.once.Do(func() {
		close(s.stop)
	})
	<-s.done
	s.mutex.Lock()
	clear(s.items)
	s.mutex.Unlock()
}
