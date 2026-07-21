package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestMemorySchemaRepoSaveDomainSchemasOnce(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	schema := testDomainSchema()

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})
	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	entry := memoryDomainSchemaByHash[schema.Hash]
	require.NotNil(t, entry)
	assert.Same(t, schema, entry.Schema)
	assert.Len(t, memoryDomainSchemaByHash, 1)
}

func TestMemorySchemaRepoSharesSchemasAcrossInstances(t *testing.T) {
	resetMemorySchemaRepoForTest()
	writer := new(MemorySchemaRepo)
	reader := new(MemorySchemaRepo)
	schema := testDomainSchema()

	writer.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	views := reader.ListDomainSchemaViews()
	require.Len(t, views, 1)
	assert.Same(t, schema, views[0].DomainVersion.Schema)
	assert.Len(t, reader.ListAppConfigSchemas(), 1)
}

func TestMemorySchemaRepoReleaseDomainSchemas(t *testing.T) {
	resetMemorySchemaRepoForTest()
	repo := new(MemorySchemaRepo)
	oldSchema := testDomainSchema()
	newSchema := testDomainSchema()
	newSchema.Hash = "pkg-hash-2"

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("demo.app", "instance-2", []*skel.DomainSchema{newSchema})
	repo.ReleaseDomainSchemas("demo.app", "instance-2")

	_, ok := memoryDomainSchemaByHash[newSchema.Hash]
	assert.False(t, ok)
	views := repo.ListDomainSchemaViews()
	require.Len(t, views, 1)
	assert.Same(t, oldSchema, views[0].DomainVersion.Schema)
	assert.True(t, views[0].DomainVersion.Main)
	assert.False(t, views[0].DomainVersion.MultiVersion)
}
