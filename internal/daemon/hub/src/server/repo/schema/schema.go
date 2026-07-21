package schema

import (
	"sync"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

// MemorySchemaRepo

type MemorySchemaRepo struct{}

type _MemoryDomainSchemaEntry struct {
	Schema   *skel.DomainSchema
	Sequence int
	OwnerIDs map[string]struct{}
}

type _MemorySchemaRef[T any] struct {
	SkelName string
	Hash     string
	Schema   T
}

type _MemorySchemaVersionState struct {
	DefaultHash    string
	MainDomainHash string
	Hashes         map[string]struct{}
}

type _MemorySchemaSnapshot struct {
	DomainSchemas    []*skel.DomainSchema
	DomainVersions   []core.DomainSchemaVersion
	DomainViews      []core.DomainSchemaView
	VineHubViews     []core.DomainSchemaView
	ConfigSchemas    []*skel.ConfigSchema
	ActorSchemas     []*skel.ActorSchema
	DataSchemas      []*skel.DataSchema
	EnumSchemas      []*skel.EnumSchema
	EventSchemas     []*skel.EventSchema
	ResourceSchemas  []*skel.ResourceSchema
	ServiceSchemas   []*skel.ServiceSchema
	TaskSchemas      []*skel.TaskSchema
	WebSchemas       []*skel.WebSchema
	ActorVersions    []core.SchemaVersion[*skel.ActorSchema]
	ConfigVersions   []core.SchemaVersion[*skel.ConfigSchema]
	DataVersions     []core.SchemaVersion[*skel.DataSchema]
	EnumVersions     []core.SchemaVersion[*skel.EnumSchema]
	EventVersions    []core.SchemaVersion[*skel.EventSchema]
	ResourceVersions []core.SchemaVersion[*skel.ResourceSchema]
	ServiceVersions  []core.SchemaVersion[*skel.ServiceSchema]
	TaskVersions     []core.SchemaVersion[*skel.TaskSchema]
	WebVersions      []core.SchemaVersion[*skel.WebSchema]
}

var (
	memoryDomainSchemaMutex          sync.RWMutex
	memoryDomainSchemaSequence       int
	memoryDomainSchemaByHash         = map[string]*_MemoryDomainSchemaEntry{}
	memoryDomainSchemaHashesByDomain = map[string]map[string]struct{}{}
	memorySchemaHashesByOwner        = map[string]map[string]struct{}{}
	memorySchemaSnapshot             = _MemorySchemaSnapshot{}
)

func (r *MemorySchemaRepo) SaveDomainSchemasJSON(ownerName string, ownerId string, schemas []skel.JSON) {
	domainSchemas := make([]*skel.DomainSchema, 0, len(schemas))
	for _, schemaJson := range schemas {
		schema := vcode.MustUnmarshalJsonS[*skel.DomainSchema](string(schemaJson))
		domainSchemas = append(domainSchemas, schema)
	}
	r.SaveDomainSchemas(ownerName, ownerId, domainSchemas)
}

func (*MemorySchemaRepo) SaveDomainSchemas(ownerName string, ownerId string, schemas []*skel.DomainSchema) {
	memoryDomainSchemaMutex.Lock()
	defer memoryDomainSchemaMutex.Unlock()

	ownerKey := memoryOwnerKey(ownerName, ownerId)
	memoryReleaseDomainSchemas(ownerKey)
	for _, schema := range schemas {
		memoryRetainDomainSchemaForOwner(ownerKey, schema)
	}
	memoryRefreshSchemaSnapshot()
}

func (*MemorySchemaRepo) ReleaseDomainSchemas(ownerName string, ownerId string) {
	memoryDomainSchemaMutex.Lock()
	defer memoryDomainSchemaMutex.Unlock()

	ownerKey := memoryOwnerKey(ownerName, ownerId)
	memoryReleaseDomainSchemas(ownerKey)
	memoryRefreshSchemaSnapshot()
}

func memoryRetainDomainSchemaForOwner(ownerKey string, schema *skel.DomainSchema) {
	entry := memoryRetainDomainSchema(schema)
	entry.OwnerIDs[ownerKey] = struct{}{}
	if memorySchemaHashesByOwner[ownerKey] == nil {
		memorySchemaHashesByOwner[ownerKey] = map[string]struct{}{}
	}
	memorySchemaHashesByOwner[ownerKey][schema.Hash] = struct{}{}
}

func memoryReleaseDomainSchemas(ownerKey string) {
	hashes := memorySchemaHashesByOwner[ownerKey]
	for hash := range hashes {
		entry := memoryDomainSchemaByHash[hash]
		delete(entry.OwnerIDs, ownerKey)
		if len(entry.OwnerIDs) == 0 {
			memoryRemoveDomainSchemaEntry(hash, entry.Schema.Domain)
		}
	}
	delete(memorySchemaHashesByOwner, ownerKey)
}

func (*MemorySchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.DomainSchemaView{}, memorySchemaSnapshot.DomainViews...)
}

func (*MemorySchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.DomainSchemaView{}, memorySchemaSnapshot.VineHubViews...)
}

func (*MemorySchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.ActorSchema]{}, memorySchemaSnapshot.ActorVersions...)
}

func (*MemorySchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.ConfigSchema]{}, memorySchemaSnapshot.ConfigVersions...)
}

func (*MemorySchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.DataSchema]{}, memorySchemaSnapshot.DataVersions...)
}

func (*MemorySchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.EnumSchema]{}, memorySchemaSnapshot.EnumVersions...)
}

func (*MemorySchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.EventSchema]{}, memorySchemaSnapshot.EventVersions...)
}

func (*MemorySchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.ResourceSchema]{}, memorySchemaSnapshot.ResourceVersions...)
}

func (*MemorySchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.ServiceSchema]{}, memorySchemaSnapshot.ServiceVersions...)
}

func (*MemorySchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.TaskSchema]{}, memorySchemaSnapshot.TaskVersions...)
}

func (*MemorySchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]core.SchemaVersion[*skel.WebSchema]{}, memorySchemaSnapshot.WebVersions...)
}

func (*MemorySchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]*skel.ConfigSchema{}, memorySchemaSnapshot.ConfigSchemas...)
}

func (*MemorySchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]*skel.ActorSchema{}, memorySchemaSnapshot.ActorSchemas...)
}

func (*MemorySchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]*skel.EnumSchema{}, memorySchemaSnapshot.EnumSchemas...)
}

func (*MemorySchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]*skel.ServiceSchema{}, memorySchemaSnapshot.ServiceSchemas...)
}

func (*MemorySchemaRepo) ListWebSchemas() []*skel.WebSchema {
	memoryDomainSchemaMutex.RLock()
	defer memoryDomainSchemaMutex.RUnlock()

	return append([]*skel.WebSchema{}, memorySchemaSnapshot.WebSchemas...)
}
