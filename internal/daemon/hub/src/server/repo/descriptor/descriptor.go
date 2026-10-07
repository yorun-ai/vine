package descriptor

import (
	"sync"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

// DescriptorRepo stores the descriptors applications register and selects the versions
// the Hub serves. The Hub binds it per application, so every reader sees the
// same state.
type DescriptorRepo struct {
	mu       sync.RWMutex
	sequence int

	byHash         map[string]*_DomainDescriptorEntry
	hashesByDomain map[string]map[string]struct{}
	hashesByOwner  map[string]map[string]struct{}

	snapshot _DescriptorSnapshot
}

type _DomainDescriptorEntry struct {
	Descriptor *skeldesc.Domain
	Sequence   int
	OwnerIDs   map[string]struct{}
}

type _DescriptorRef[T any] struct {
	SkelName   string
	Hash       string
	Descriptor T
}

type _DescriptorVersionState struct {
	DefaultHash    string
	MainDomainHash string
	Hashes         map[string]struct{}
}

func (r *DescriptorRepo) SaveDomainDescriptors(ownerName string, ownerId string, descriptors []*skeldesc.Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, descriptor := range descriptors {
		if err := skel.ValidateDescriptor(descriptor); err != nil {
			panic(err)
		}
	}

	ownerKey := descriptorOwnerKey(ownerName, ownerId)
	r.releaseDomainDescriptors(ownerKey)
	for _, descriptor := range descriptors {
		r.retainDomainDescriptorForOwner(ownerKey, descriptor)
	}
	r.refreshSnapshot()
}

func (r *DescriptorRepo) ReleaseDomainDescriptors(ownerName string, ownerId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.releaseDomainDescriptors(descriptorOwnerKey(ownerName, ownerId))
	r.refreshSnapshot()
}

func (r *DescriptorRepo) retainDomainDescriptorForOwner(ownerKey string, descriptor *skeldesc.Domain) {
	if r.hashesByOwner == nil {
		r.hashesByOwner = map[string]map[string]struct{}{}
	}
	entry := r.retainDomainDescriptor(descriptor)
	entry.OwnerIDs[ownerKey] = struct{}{}
	if r.hashesByOwner[ownerKey] == nil {
		r.hashesByOwner[ownerKey] = map[string]struct{}{}
	}
	r.hashesByOwner[ownerKey][descriptor.Hash] = struct{}{}
}

func (r *DescriptorRepo) releaseDomainDescriptors(ownerKey string) {
	for hash := range r.hashesByOwner[ownerKey] {
		entry := r.byHash[hash]
		delete(entry.OwnerIDs, ownerKey)
		if len(entry.OwnerIDs) == 0 {
			r.removeDomainDescriptorEntry(hash, entry.Descriptor.Name)
		}
	}
	delete(r.hashesByOwner, ownerKey)
}

func (r *DescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Domains.Views()
}

func (r *DescriptorRepo) ListVineHubDescriptorViews() []core.DomainDescriptorView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Domains.VineHubViews()
}

func (r *DescriptorRepo) ListActorDescriptorVersions() []core.DescriptorVersion[*skeldesc.Actor] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Actors.Versions()
}

func (r *DescriptorRepo) ListConfigDescriptorVersions() []core.DescriptorVersion[*skeldesc.Config] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Configs.Versions()
}

func (r *DescriptorRepo) ListDataDescriptorVersions() []core.DescriptorVersion[*skeldesc.Data] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Data.Versions()
}

func (r *DescriptorRepo) ListEnumDescriptorVersions() []core.DescriptorVersion[*skeldesc.Enum] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Enums.Versions()
}

func (r *DescriptorRepo) ListEventDescriptorVersions() []core.DescriptorVersion[*skeldesc.Event] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Events.Versions()
}

func (r *DescriptorRepo) ListResourceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Resource] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Resources.Versions()
}

func (r *DescriptorRepo) ListServiceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Service] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Services.Versions()
}

func (r *DescriptorRepo) ListTaskDescriptorVersions() []core.DescriptorVersion[*skeldesc.Task] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Tasks.Versions()
}

func (r *DescriptorRepo) ListWebDescriptorVersions() []core.DescriptorVersion[*skeldesc.Web] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Versions()
}

func (r *DescriptorRepo) ListAppConfigDescriptors() []*skeldesc.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Configs.Selected()
}

func (r *DescriptorRepo) ListActorDescriptors() []*skeldesc.Actor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Actors.Selected()
}

func (r *DescriptorRepo) ListEnumDescriptors() []*skeldesc.Enum {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Enums.Selected()
}

func (r *DescriptorRepo) ListServiceDescriptors() []*skeldesc.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Services.Selected()
}

func (r *DescriptorRepo) ListWebDescriptors() []*skeldesc.Web {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Selected()
}

// GetWebDescriptor returns the selected descriptor with the Skel name, or nil when the
// Hub does not serve it.
func (r *DescriptorRepo) GetWebDescriptor(skelName string) *skeldesc.Web {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Get(skelName)
}

// ListAppConfigTypeDescriptors returns config declarations and their value types from one snapshot.
func (r *DescriptorRepo) ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snapshot.Configs.Selected(), r.snapshot.Enums.Selected(), r.snapshot.Data.Selected()
}
