package redisserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"math"
	"net"
	"time"

	"go.yorun.ai/vine/internal/app"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/comp/redisserver/embedded"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vpre"
)

type Server struct {
	app.BaseComponent

	Context    context.Context         `inject:""`
	Option     *flag.Flag              `inject:""`
	InprocFlag *app.InternalInprocFlag `inject:""`

	store          _RedisStore
	serverPassword string
}

func (s *Server) DIInit() {
	s.serverPassword = newServerPassword()
	s.store = embedded.NewStore(s.Option.RedisListen, s.InprocFlag.Enabled, s.serverPassword)
	s.store.Start()
	s.store.InitRevision()
	if s.InprocFlag.Enabled {
		hubredis.SetInprocServer(s)
	}
}

func (s *Server) AfterAppStop() {
	if hubredis.InprocServer() == s {
		hubredis.SetInprocServer(nil)
	}
	s.store.Stop()
	s.serverPassword = ""
}

func newServerPassword() string {
	random := make([]byte, 32)
	_, err := rand.Read(random)
	vpre.CheckNilError(err, "generate redis server password failed")
	return base64.RawURLEncoding.EncodeToString(random)
}

func (s *Server) DialInproc(ctx context.Context) (net.Conn, error) {
	store, ok := s.store.(*embedded.Store)
	vpre.Check(ok, "redis inproc and embedded protocol modes require embedded store")
	return store.DialInproc(ctx)
}

func (s *Server) Set(key string, value string) {
	s.store.Set(key, value)
}

func (s *Server) Get(key string) (string, bool) {
	return s.store.Get(key)
}

func (s *Server) TTL(key string) int {
	return s.store.TTL(key)
}

func (s *Server) Scan(pattern string) []string {
	return s.store.Scan(pattern)
}

func (s *Server) Incr(key string) (int64, error) {
	return s.store.Incr(key)
}

func (s *Server) Del(key string) bool {
	return s.store.Del(key)
}

func (s *Server) Expire(key string, seconds int) bool {
	return s.store.Expire(key, seconds)
}

func (s *Server) Publish(channel string, message string) int {
	return s.store.Publish(channel, message)
}

func (s *Server) SetEphemeral(key string, value string, ttl time.Duration) {
	s.store.SetWithTTL(key, value, ttl)
}

func (s *Server) SetAndNotify(key string, value string) {
	s.store.SetAndNotify(key, value)
}

func (s *Server) SetEphemeralAndNotify(key string, value string, ttl time.Duration) {
	s.store.SetWithTTLAndNotify(key, value, ttl)
}

func (s *Server) DeleteAndNotify(key string) {
	s.store.DeleteAndNotify(key)
}

func (s *Server) applyAndNotify(operations []hubredis.NotifyOperation) {
	s.store.ApplyAndNotify(operations)
}

// KeepEphemeral refreshes the TTL for an existing ephemeral key. It returns false when the key is missing.
func (s *Server) KeepEphemeral(key string, ttl time.Duration) bool {
	return s.store.Keep(key, ttl)
}

func (s *Server) KeepLease(key string, member string, ttl time.Duration) {
	score := float64(timeNow().Add(ttl).UnixMilli())
	s.store.ZAdd(key, score, member)
}

func (s *Server) PopExpiredLeases(key string, limit int) []string {
	return s.store.ZPopRangeByScore(key, math.Inf(-1), float64(timeNow().UnixMilli()), limit)
}

func (s *Server) RemoveLease(key string, member string) bool {
	return s.store.ZRem(key, member)
}
