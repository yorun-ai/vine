package desc

import (
	"context"
	"encoding/json/v2"
	"maps"
	"slices"
	"strings"
	"time"

	skeldesc "go.yorun.ai/skel/descriptor"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/util/vmap"
)

// Snapshot combines current admission metadata with the last completed paired
// API service/type snapshot. Maps belong to the caller; descriptors are immutable.
// MissingKeys contains referenced definitions that have not been published yet.
type Snapshot struct {
	Actors      map[string]*skeldesc.Actor
	Resources   map[string]*skeldesc.Resource
	Webs        map[string]*skeldesc.Web
	Services    map[string]*skeldesc.Service
	Data        map[string]*skeldesc.Data
	Enums       map[string]*skeldesc.Enum
	MissingKeys []string
	// Syncing reports that the paired service/type snapshot trails admission state.
	Syncing bool
}

// Snapshot returns the contracts retained for API inspection, without making
// callers read Hub or maintain another descriptor cache.
func (m *Manager) Snapshot() Snapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	snapshot := Snapshot{
		Actors:    maps.Clone(m.actorsBySkelName),
		Resources: maps.Clone(m.resourcesBySkelName),
		Webs:      maps.Clone(m.websBySkelName),
		Services:  maps.Clone(m.typeServices),
		Syncing:   m.generation != m.syncedGeneration,
		Data:      map[string]*skeldesc.Data{}, Enums: map[string]*skeldesc.Enum{},
	}
	for _, key := range vmap.SortedKeys(m.typesByKey) {
		entry := m.typesByKey[key]
		switch {
		case entry.data != nil:
			snapshot.Data[entry.name] = entry.data
		case entry.enum != nil:
			snapshot.Enums[entry.name] = entry.enum
		default:
			snapshot.MissingKeys = append(snapshot.MissingKeys, key)
		}
	}
	return snapshot
}

type _TypeDescriptor struct {
	name string
	kind string
	data *skeldesc.Data
	enum *skeldesc.Enum
}

func newTypeDescriptor(key string) *_TypeDescriptor {
	parts := strings.SplitN(key, ":", 3)
	return new(_TypeDescriptor{kind: parts[1], name: parts[2]})
}

func (entry *_TypeDescriptor) setValue(value string) error {
	entry.data = nil
	entry.enum = nil
	if value == "" {
		return nil
	}
	switch entry.kind {
	case "data":
		return json.Unmarshal([]byte(value), &entry.data)
	case "enum":
		return json.Unmarshal([]byte(value), &entry.enum)
	}
	return nil
}

func (entry *_TypeDescriptor) members() []*skeldesc.Member {
	switch {
	case entry.data != nil:
		return entry.data.Members
	default:
		return nil
	}
}

type _TypeSubscription struct {
	ctx    context.Context
	cancel context.CancelFunc
	keys   []string
	watch  hubwatch.Subscription
}

// selectTypes is a pure reachability pass. Missing definitions stay in the set
// so they remain subscribed. Entries are immutable and can be shared by snapshots.
func selectTypes(services map[string]*skeldesc.Service, values map[string]*_TypeDescriptor) map[string]*_TypeDescriptor {
	wanted := map[string]*_TypeDescriptor{}
	var visit func(kind *skeldesc.Type)
	visit = func(kind *skeldesc.Type) {
		if kind == nil {
			return
		}
		key := ""
		switch kind.Kind {
		case skeldesc.TypeKindData:
			key = watched.FormatDescriptorDataKey(kind.SkelName)
		case skeldesc.TypeKindEnum:
			key = watched.FormatDescriptorEnumKey(kind.SkelName)
		}
		if key != "" && wanted[key] == nil {
			entry := values[key]
			if entry == nil {
				entry = newTypeDescriptor(key)
			}
			wanted[key] = entry
			for _, member := range entry.members() {
				visit(member.Type)
			}
		}
		visit(kind.Element)
		visit(kind.Key)
		visit(kind.Value)
		for _, argument := range kind.TypeArguments {
			visit(argument)
		}
	}
	for _, service := range services {
		for _, method := range service.Methods {
			for _, argument := range method.Arguments {
				visit(argument.Type)
			}
			visit(method.ResultType)
		}
	}
	return wanted
}

func (m *Manager) requestTypeSyncLocked() {
	m.generation++
	select {
	case m.syncWake <- struct{}{}:
	default:
	}
}

// runTypeSync serializes and coalesces refresh work. Only transport failures
// back off; superseded work immediately retries from the latest in-memory roots.
func (m *Manager) runTypeSync() {
	defer close(m.syncDone)
	for {
		select {
		case <-m.watchContext.Done():
			return
		case <-m.syncWake:
		}
		delay := 100 * time.Millisecond
		for m.watchContext.Err() == nil {
			err := m.syncTypes()
			if err != nil {
				if m.watchContext.Err() != nil {
					return
				}
				descLogger.Warn("descriptor type synchronization failed", "error", err)
				timer := time.NewTimer(delay)
				select {
				case <-m.watchContext.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				delay = min(2*delay, 5*time.Second)
				continue
			}
			delay = 100 * time.Millisecond
			m.mutex.RLock()
			current := m.syncedGeneration == m.generation
			m.mutex.RUnlock()
			if current {
				break
			}
		}
	}
}

// syncTypes builds a candidate without holding the cache lock. The active
// subscription and published snapshot survive failures and superseded attempts.
func (m *Manager) syncTypes() error {
	m.mutex.RLock()
	if m.stopped || m.syncedGeneration == m.generation {
		m.mutex.RUnlock()
		return nil
	}
	generation := m.generation
	services := maps.Clone(m.servicesBySkelName)
	values := maps.Clone(m.typeInputs)
	previous := m.typeSubscription
	m.mutex.RUnlock()

	candidate := previous
	owned := false
	defer func() {
		if owned {
			candidate.cancel()
		}
	}()
	var wanted map[string]*_TypeDescriptor
	for {
		if err := m.watchContext.Err(); err != nil {
			return err
		}
		wanted = selectTypes(services, values)
		keys := vmap.SortedKeys(wanted)
		if len(keys) == 0 {
			if owned {
				candidate.cancel()
				owned = false
			}
			candidate = nil
			break
		}
		if candidate != nil && slices.Equal(keys, candidate.keys) {
			break
		}
		next := new(_TypeSubscription{keys: keys})
		next.ctx, next.cancel = context.WithCancel(m.watchContext)
		loaded, subscription, err := m.Watch.LoadKeysAndSubscribe(next.ctx, keys, func(event hubwatch.Event) { m.handleTypeEvent(next, event) })
		if err != nil {
			next.cancel()
			return err
		}
		next.watch = subscription
		if owned {
			candidate.cancel()
		}
		candidate, owned = next, true
		for _, key := range keys {
			entry := newTypeDescriptor(key)
			if err := entry.setValue(loaded[key]); err != nil {
				return err
			}
			values[key] = entry
		}
		m.mutex.RLock()
		superseded := m.generation != generation || m.stopped
		m.mutex.RUnlock()
		if superseded {
			return nil
		}
		// The attached snapshot can introduce or remove transitive dependencies.
	}

	m.mutex.Lock()
	if m.stopped || m.watchContext.Err() != nil || m.generation != generation {
		m.mutex.Unlock()
		return nil
	}
	m.typeServices = services
	m.typesByKey = wanted
	m.typeInputs = maps.Clone(wanted)
	m.typeSubscription = candidate
	m.syncedGeneration = generation
	owned = false
	m.mutex.Unlock()
	if candidate != previous {
		if candidate != nil {
			candidate.watch.Start()
		}
		if previous != nil {
			previous.cancel()
		}
	}
	return nil
}

func (m *Manager) handleTypeEvent(subscription *_TypeSubscription, event hubwatch.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.stopped || m.typeSubscription != subscription || subscription.ctx.Err() != nil {
		return
	}
	if m.typeInputs[event.Key] == nil {
		return
	}
	entry := newTypeDescriptor(event.Key)
	value := event.Value
	if event.Kind == hubwatch.EventKindDelete {
		value = ""
	}
	if err := entry.setValue(value); err != nil {
		descLogger.Warn("descriptor type update is invalid", "key", event.Key, "error", err)
		return
	}
	m.typeInputs[event.Key] = entry
	m.requestTypeSyncLocked()
}

// AfterAppStop cancels in-flight refreshes and waits for their worker to exit.
func (m *Manager) AfterAppStop() {
	m.watchCancel()
	m.mutex.Lock()
	m.stopped = true
	if m.typeSubscription != nil {
		m.typeSubscription.cancel()
	}
	m.mutex.Unlock()
	<-m.syncDone
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.typeSubscription = nil
	clear(m.typesByKey)
	clear(m.typeInputs)
	clear(m.typeServices)
	clear(m.admissionObservers)
}
