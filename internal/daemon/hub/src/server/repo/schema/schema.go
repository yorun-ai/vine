package schema

import (
	"sync"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// SchemaRepo stores the schemas applications register and selects the versions
// the Hub serves. The Hub binds it per application, so every reader sees the
// same state.
type SchemaRepo struct {
	mu       sync.RWMutex
	sequence int

	byHash         map[string]*_DomainSchemaEntry
	hashesByDomain map[string]map[string]struct{}
	hashesByOwner  map[string]map[string]struct{}

	snapshot _SchemaSnapshot
}

type _DomainSchemaEntry struct {
	Schema   *skel.DomainSchema
	Sequence int
	OwnerIDs map[string]struct{}
}

type _SchemaRef[T any] struct {
	SkelName string
	Hash     string
	Schema   T
}

type _SchemaVersionState struct {
	DefaultHash    string
	MainDomainHash string
	Hashes         map[string]struct{}
}

func (r *SchemaRepo) SaveDomainSchemasJSON(ownerName string, ownerId string, schemas []skel.JSON) {
	domainSchemas := make([]*skel.DomainSchema, 0, len(schemas))
	for _, schemaJson := range schemas {
		schema := vcode.MustUnmarshalJsonS[*skel.DomainSchema](string(schemaJson))
		domainSchemas = append(domainSchemas, schema)
	}
	r.SaveDomainSchemas(ownerName, ownerId, domainSchemas)
}

func (r *SchemaRepo) SaveDomainSchemas(ownerName string, ownerId string, schemas []*skel.DomainSchema) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerKey := schemaOwnerKey(ownerName, ownerId)
	r.releaseDomainSchemas(ownerKey)
	for _, schema := range schemas {
		r.retainDomainSchemaForOwner(ownerKey, schema)
	}
	r.refreshSnapshot()
}

func (r *SchemaRepo) ReleaseDomainSchemas(ownerName string, ownerId string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.releaseDomainSchemas(schemaOwnerKey(ownerName, ownerId))
	r.refreshSnapshot()
}

func (r *SchemaRepo) retainDomainSchemaForOwner(ownerKey string, schema *skel.DomainSchema) {
	if r.hashesByOwner == nil {
		r.hashesByOwner = map[string]map[string]struct{}{}
	}
	entry := r.retainDomainSchema(schema)
	entry.OwnerIDs[ownerKey] = struct{}{}
	if r.hashesByOwner[ownerKey] == nil {
		r.hashesByOwner[ownerKey] = map[string]struct{}{}
	}
	r.hashesByOwner[ownerKey][schema.Hash] = struct{}{}
}

func (r *SchemaRepo) releaseDomainSchemas(ownerKey string) {
	for hash := range r.hashesByOwner[ownerKey] {
		entry := r.byHash[hash]
		delete(entry.OwnerIDs, ownerKey)
		if len(entry.OwnerIDs) == 0 {
			r.removeDomainSchemaEntry(hash, entry.Schema.Domain)
		}
	}
	delete(r.hashesByOwner, ownerKey)
}

func (r *SchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Domains.Views()
}

func (r *SchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Domains.VineHubViews()
}

func (r *SchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Actors.Versions()
}

func (r *SchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Configs.Versions()
}

func (r *SchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Data.Versions()
}

func (r *SchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Enums.Versions()
}

func (r *SchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Events.Versions()
}

func (r *SchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Resources.Versions()
}

func (r *SchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Services.Versions()
}

func (r *SchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Tasks.Versions()
}

func (r *SchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Versions()
}

func (r *SchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Configs.Selected()
}

func (r *SchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Actors.Selected()
}

func (r *SchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Enums.Selected()
}

func (r *SchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Services.Selected()
}

func (r *SchemaRepo) ListWebSchemas() []*skel.WebSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Selected()
}

// GetWebSchema returns the selected schema with the Skel name, or nil when the
// Hub does not serve it.
func (r *SchemaRepo) GetWebSchema(skelName string) *skel.WebSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshot.Webs.Get(skelName)
}
