package desc

import (
	"context"
	"maps"

	skeldesc "go.yorun.ai/skel/descriptor"
)

// AdmissionSnapshot contains immutable contracts needed to watch auth,
// permission, and resource-check endpoints. Its maps belong to the observer.
type AdmissionSnapshot struct {
	Actors    map[string]*skeldesc.Actor
	Resources map[string]*skeldesc.Resource
}

type _AdmissionObserver struct {
	ctx    context.Context
	handle func(AdmissionSnapshot)
}

// ObserveAdmission delivers the initial snapshot synchronously and subsequent
// changes in order until ctx is canceled. Callbacks run outside the cache lock.
func (m *Manager) ObserveAdmission(ctx context.Context, handle func(AdmissionSnapshot)) {
	m.admissionMutex.Lock()
	defer m.admissionMutex.Unlock()
	m.mutex.Lock()
	if ctx.Err() != nil || m.stopped {
		m.mutex.Unlock()
		return
	}
	observer := new(_AdmissionObserver{ctx: ctx, handle: handle})
	m.admissionObservers[observer] = struct{}{}
	snapshot := m.admissionSnapshotLocked()
	m.mutex.Unlock()
	context.AfterFunc(ctx, func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		delete(m.admissionObservers, observer)
	})
	if ctx.Err() == nil {
		handle(snapshot)
	}
}

func (m *Manager) admissionSnapshotLocked() AdmissionSnapshot {
	return AdmissionSnapshot{Actors: maps.Clone(m.actorsBySkelName), Resources: maps.Clone(m.resourcesBySkelName)}
}

// admissionMutex orders mutation and delivery; mutex only protects cached data.
func (m *Manager) notifyAdmission() {
	m.mutex.RLock()
	observers := make([]*_AdmissionObserver, 0, len(m.admissionObservers))
	for observer := range m.admissionObservers {
		observers = append(observers, observer)
	}
	snapshot := m.admissionSnapshotLocked()
	m.mutex.RUnlock()
	for _, observer := range observers {
		if observer.ctx.Err() == nil {
			observer.handle(AdmissionSnapshot{Actors: maps.Clone(snapshot.Actors), Resources: maps.Clone(snapshot.Resources)})
		}
	}
}
