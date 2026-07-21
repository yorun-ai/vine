package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestMemorySchemaRepoListAppConfigSchemas(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	schema := testDomainSchema()
	schema.Configs[0].Description = "Main config description"
	schema.Configs[0].Members = []*skel.MemberSchema{
		{Name: "title", Description: "Page title"},
	}

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	schemas := repo.ListAppConfigSchemas()

	require.Len(t, schemas, 1)
	assert.Same(t, schema.Configs[0], schemas[0])
}

func TestMemorySchemaRepoListEnumSchemas(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	schema := testDomainSchema()
	schema.Enums = []*skel.EnumSchema{{
		Name:     "UserStatus",
		SkelName: "demo.user.UserStatus",
		Items: []*skel.EnumItemSchema{
			{Name: "ACTIVE", Description: "启用"},
		},
	}}

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	schemas := repo.ListEnumSchemas()

	require.Len(t, schemas, 1)
	assert.Same(t, schema.Enums[0], schemas[0])
}

func TestMemorySchemaRepoListsLatestDomainSchemaByDomain(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	oldSchema := testDomainSchema()
	newSchema := testDomainSchema()
	newSchema.Hash = "pkg-hash-2"
	newSchema.Actors = []*skel.ActorSchema{
		{
			Name:     "UserActor",
			SkelName: "demo.user.UserActor",
			Hash:     "actor-hash-2",
			Vias:     []skel.ActorVia{skel.ActorViaClient, skel.ActorViaAgent},
		},
	}

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("demo.app", "instance-2", []*skel.DomainSchema{newSchema})

	actors := repo.ListActorSchemas()

	require.Len(t, actors, 1)
	assert.Equal(t, "demo.user.UserActor", actors[0].SkelName)
	assert.Len(t, memoryDomainSchemaByHash, 2)
}
