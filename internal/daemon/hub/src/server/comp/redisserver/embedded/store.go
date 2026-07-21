package embedded

import (
	"encoding/json"
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/redcon"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/logger"
	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
	"go.yorun.ai/vine/util/vpre"
)

type Store struct {
	listen   string
	inproc   bool
	items    map[string]_Item
	zsets    map[string]map[string]float64
	mutex    sync.Mutex
	scans    map[uint64]_ScanCursor
	nextScan uint64
	pubsub   redcon.PubSub
	server   *redcon.Server
	listener net.Listener
	endpoint string

	serverPassword string
}

type _Item struct {
	value    string
	expireAt time.Time
}

func NewStore(listen string, inproc bool, serverPassword string) *Store {
	vpre.CheckNotEmpty(serverPassword, "redis server password is empty")
	return &Store{
		listen:         listen,
		inproc:         inproc,
		items:          map[string]_Item{},
		zsets:          map[string]map[string]float64{},
		scans:          map[uint64]_ScanCursor{},
		serverPassword: serverPassword,
	}
}

func (s *Store) Start() {
	if s.inproc {
		listener := newInprocListener()
		s.listener = listener
		s.server = redcon.NewServer(listener.Addr().String(), s.handleCommand, nil, nil)
		go s.serve(listener)
		s.endpoint = hubredis.RedisInprocEndpoint
		logger.Info("vine.hub embedded redis server started", "mode", "inproc", "endpoint", s.endpoint)
		return
	}

	listener, err := net.Listen("tcp", s.listen)
	vpre.CheckNilError(err, "start redis listener failed")
	s.listener = listener
	s.server = redcon.NewServer(listener.Addr().String(), s.handleCommand, nil, nil)
	go s.serve(listener)
	if err := waitRedisReady(listener.Addr().String(), 2*time.Second); err != nil {
		s.Stop()
		vpre.CheckNilError(err, "redis server start failed")
	}
	s.endpoint = redisEndpoint(listener.Addr().String())
	logger.Info("vine.hub embedded redis server started", "addr", listener.Addr().String(), "endpoint", s.endpoint)
}

func (s *Store) Stop() {
	if s.server != nil {
		_ = s.server.Close()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *Store) Endpoint() string {
	return s.endpoint
}

func (s *Store) ListenAddr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Store) serve(listener net.Listener) {
	err := s.server.Serve(listener)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		logger.Error("vine.hub redis server failed", "error", err)
	}
}

func (s *Store) Set(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.items[key] = _Item{value: value}
}

func (s *Store) Get(key string) (string, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.getLocked(key, timeNow())
}

func (s *Store) TTL(key string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	item, ok := s.items[key]
	if !ok {
		return -2
	}
	now := timeNow()
	if item.expired(now) {
		delete(s.items, key)
		return -2
	}
	if item.expireAt.IsZero() {
		return -1
	}
	ttl := int(item.expireAt.Sub(now).Seconds())
	if ttl < 0 {
		return -2
	}
	return ttl
}

func (s *Store) Scan(pattern string) []string {
	keys, err := s.scanAllKeys(_ScanOption{
		match: pattern,
		count: scanDefaultCount,
	})
	ex.PanicIfError(err)
	return keys
}

func (s *Store) Incr(key string) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := timeNow()
	item, ok := s.getItemLocked(key, now)
	var value int64
	if ok {
		parsed, err := strconv.ParseInt(item.value, 10, 64)
		if err != nil {
			return 0, err
		}
		value = parsed
	}
	value++
	item.value = strconv.FormatInt(value, 10)
	s.items[key] = item
	return value, nil
}

func (s *Store) Del(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, ok := s.getItemLocked(key, timeNow())
	if ok {
		delete(s.items, key)
	}
	return ok
}

func (s *Store) Expire(key string, seconds int) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	item, ok := s.getItemLocked(key, timeNow())
	if !ok {
		return false
	}
	item.expireAt = timeNow().Add(time.Duration(seconds) * time.Second)
	s.items[key] = item
	return true
}

func (s *Store) Publish(channel string, message string) int {
	return s.pubsub.Publish(channel, message)
}

func (s *Store) Subscribe(conn redcon.Conn, channel string, pattern bool) {
	if pattern {
		s.pubsub.Psubscribe(conn, channel)
		return
	}
	s.pubsub.Subscribe(conn, channel)
}

func (s *Store) SetWithTTL(key string, value string, ttl time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.items[key] = _Item{value: value, expireAt: timeNow().Add(time.Duration(ttlSeconds(ttl)) * time.Second)}
}

func (s *Store) InitRevision() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.getLocked(hubredis.RevisionKey, timeNow()); ok {
		return
	}
	s.items[hubredis.RevisionKey] = _Item{value: strconv.FormatInt(timeNow().UnixMilli(), 10)}
}

func (s *Store) SetAndNotify(key string, value string) {
	s.setAndNotify(key, value, time.Time{})
}

func (s *Store) SetWithTTLAndNotify(key string, value string, ttl time.Duration) {
	s.setAndNotify(key, value, timeNow().Add(time.Duration(ttlSeconds(ttl))*time.Second))
}

func (s *Store) setAndNotify(key string, value string, expireAt time.Time) {
	s.mutex.Lock()
	oldValue, ok := s.getLocked(key, timeNow())
	if ok && oldValue == value {
		if !expireAt.IsZero() {
			s.items[key] = _Item{value: value, expireAt: expireAt}
		}
		s.mutex.Unlock()
		return
	}
	s.items[key] = _Item{value: value, expireAt: expireAt}
	revision := s.nextRevisionLocked()
	s.mutex.Unlock()

	s.publish(key, hubredis.Event{
		Revision: revision,
		Kind:     hubredis.EventKindUpsert,
		Key:      key,
		Value:    value,
	})
}

func (s *Store) DeleteAndNotify(key string) {
	s.mutex.Lock()
	_, ok := s.getItemLocked(key, timeNow())
	if !ok {
		s.mutex.Unlock()
		return
	}
	delete(s.items, key)
	revision := s.nextRevisionLocked()
	s.mutex.Unlock()

	s.publish(key, hubredis.Event{
		Revision: revision,
		Kind:     hubredis.EventKindDelete,
		Key:      key,
	})
}

func (s *Store) ApplyAndNotify(operations []hubredis.NotifyOperation) {
	now := timeNow()
	events := make([]hubredis.Event, 0, len(operations))

	s.mutex.Lock()
	for _, operation := range operations {
		if operation.Delete {
			if _, ok := s.getItemLocked(operation.Key, now); !ok {
				continue
			}
			delete(s.items, operation.Key)
			events = append(events, hubredis.Event{
				Kind: hubredis.EventKindDelete,
				Key:  operation.Key,
			})
			continue
		}

		oldValue, ok := s.getLocked(operation.Key, now)
		if ok && oldValue == operation.Value {
			continue
		}
		s.items[operation.Key] = _Item{value: operation.Value}
		events = append(events, hubredis.Event{
			Kind:  hubredis.EventKindUpsert,
			Key:   operation.Key,
			Value: operation.Value,
		})
	}
	if len(events) == 0 {
		s.mutex.Unlock()
		return
	}
	revision := s.nextRevisionLocked()
	s.mutex.Unlock()

	for _, event := range events {
		event.Revision = revision
		s.publish(event.Key, event)
	}
}

func (s *Store) Keep(key string, ttl time.Duration) bool {
	return s.Expire(key, ttlSeconds(ttl))
}

func (s *Store) ZAdd(key string, score float64, member string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	zset := s.zsets[key]
	if zset == nil {
		zset = map[string]float64{}
		s.zsets[key] = zset
	}
	zset[member] = score
}

func (s *Store) ZPopRangeByScore(key string, min float64, max float64, limit int) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	items := s.zRangeByScoreLocked(key, min, max, limit)
	zset := s.zsets[key]
	members := make([]string, 0, len(items))
	for _, item := range items {
		delete(zset, item.member)
		members = append(members, item.member)
	}
	if len(zset) == 0 {
		delete(s.zsets, key)
	}
	return members
}

func (s *Store) zRangeByScoreLocked(key string, min float64, max float64, limit int) []_ZSetItem {
	zset := s.zsets[key]
	if len(zset) == 0 {
		return nil
	}
	items := make([]_ZSetItem, 0)
	for member, score := range zset {
		if score < min || score > max {
			continue
		}
		items = append(items, _ZSetItem{member: member, score: score})
	}
	sort.Slice(items, func(i int, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score < items[j].score
		}
		return items[i].member < items[j].member
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *Store) ZRem(key string, member string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	zset := s.zsets[key]
	if len(zset) == 0 {
		return false
	}
	if _, ok := zset[member]; !ok {
		return false
	}
	delete(zset, member)
	if len(zset) == 0 {
		delete(s.zsets, key)
	}
	return true
}

func (s *Store) publish(key string, event hubredis.Event) {
	payload, err := json.Marshal(event)
	ex.PanicIfError(err)
	s.Publish(key, string(payload))
}

func (s *Store) nextRevisionLocked() uint64 {
	item, _ := s.getItemLocked(hubredis.RevisionKey, timeNow())
	revision, err := strconv.ParseUint(item.value, 10, 64)
	if err != nil {
		revision = 0
	}
	revision++
	item.value = strconv.FormatUint(revision, 10)
	s.items[hubredis.RevisionKey] = item
	return revision
}

func (s *Store) getLocked(key string, now time.Time) (string, bool) {
	item, ok := s.getItemLocked(key, now)
	if !ok {
		return "", false
	}
	return item.value, true
}

func (s *Store) getItemLocked(key string, now time.Time) (_Item, bool) {
	item, ok := s.items[key]
	if !ok {
		return _Item{}, false
	}
	if item.expired(now) {
		delete(s.items, key)
		return _Item{}, false
	}
	return item, true
}

type _ZSetItem struct {
	member string
	score  float64
}

func (i _Item) expired(now time.Time) bool {
	return !i.expireAt.IsZero() && !now.Before(i.expireAt)
}

func ttlSeconds(ttl time.Duration) int {
	vpre.Check(ttl > 0, "ttl must be greater than 0")
	seconds := int(ttl / time.Second)
	if ttl%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		seconds = 1
	}
	return seconds
}
