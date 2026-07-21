package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestMemorySchemaRepoSaveDomainSchemas(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	oldSchema := testDomainSchema()
	newSchema := testDomainSchema()
	newSchema.Hash = "pkg-hash-2"

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("demo.app", "instance-2", []*skel.DomainSchema{newSchema})

	views := repo.ListDomainSchemaViews()

	require.Len(t, views, 2)
	assert.Same(t, newSchema, views[0].DomainVersion.Schema)
	assert.True(t, views[0].DomainVersion.Main)
	assert.True(t, views[0].DomainVersion.MultiVersion)
	assert.Same(t, oldSchema, views[1].DomainVersion.Schema)
	assert.False(t, views[1].DomainVersion.Main)
	assert.True(t, views[1].DomainVersion.MultiVersion)
	require.Len(t, views[1].Actors, 1)
	require.Len(t, views[1].Configs, 1)
	require.Len(t, views[1].Data, 1)
	require.Len(t, views[1].Events, 1)
	require.Len(t, views[1].Services, 1)
	require.Len(t, views[1].Tasks, 1)
	require.Len(t, views[1].Webs, 1)
}

func TestMemorySchemaRepoListServiceSchemaVersions(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	oldSchema := testDomainSchema()
	oldSchema.Hash = "domain-old"
	oldSchema.Services = []*skel.ServiceSchema{
		{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-old"},
		{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-b"},
	}
	mainSchema := testDomainSchema()
	mainSchema.Hash = "domain-main"
	mainSchema.Services = []*skel.ServiceSchema{
		{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-main"},
	}
	crossSchema := testDomainSchema()
	crossSchema.Hash = "domain-cross"
	crossSchema.Services = []*skel.ServiceSchema{
		{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-a"},
	}

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("demo.app", "instance-2", []*skel.DomainSchema{crossSchema})
	repo.SaveDomainSchemas("demo.app", "instance-3", []*skel.DomainSchema{mainSchema})

	versions := repo.ListServiceSchemaVersions()

	require.Len(t, versions, 4)
	assert.Equal(t, "demo.user.ChangedService", versions[0].SkelName)
	assert.True(t, versions[0].Main)
	assert.Equal(t, "changed-service-main", versions[0].SchemaHash)
	assert.Equal(t, "changed-service-main", versions[0].MainSchemaHash)
	assert.True(t, versions[0].MultiVersion)
	assert.Equal(t, "demo.user.ChangedService", versions[1].SkelName)
	assert.False(t, versions[1].Main)
	assert.Equal(t, "changed-service-old", versions[1].SchemaHash)
	assert.Equal(t, "changed-service-main", versions[1].MainSchemaHash)
	assert.True(t, versions[1].MultiVersion)
	assert.Equal(t, "demo.user.RemovedService", versions[2].SkelName)
	assert.True(t, versions[2].Main)
	assert.Equal(t, "removed-service-a", versions[2].SchemaHash)
	assert.Equal(t, "removed-service-a", versions[2].MainSchemaHash)
	assert.True(t, versions[2].MultiVersion)
	assert.Equal(t, "demo.user.RemovedService", versions[3].SkelName)
	assert.False(t, versions[3].Main)
	assert.Equal(t, "removed-service-b", versions[3].SchemaHash)
	assert.Equal(t, "removed-service-a", versions[3].MainSchemaHash)
	assert.True(t, versions[3].MultiVersion)
}

func TestMemorySchemaRepoListsVineHubSchemaViews(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	domainSchema := &skel.DomainSchema{
		Domain: "vine.hub",
		Hash:   "hub-domain-hash",
		Actors: []*skel.ActorSchema{{
			Name:     "AdminActor",
			SkelName: "vine.hub.AdminActor",
			Hash:     "admin-actor-hash",
		}},
		Services: []*skel.ServiceSchema{{
			Name:     "SkeletonService",
			SkelName: "vine.hub.SkeletonService",
			Hash:     "skeleton-service-hash",
		}},
	}

	repo.SaveDomainSchemas("vine.hub.inproc", "registered", []*skel.DomainSchema{domainSchema})

	assert.Empty(t, repo.ListActorSchemaVersions())
	assert.Empty(t, repo.ListServiceSchemaVersions())

	views := repo.ListVineHubSchemaViews()
	require.Len(t, views, 1)
	require.Len(t, views[0].Actors, 1)
	assert.Equal(t, "vine.hub.AdminActor", views[0].Actors[0].SkelName)
	require.Len(t, views[0].Services, 1)
	assert.Equal(t, "vine.hub.SkeletonService", views[0].Services[0].SkelName)
}
