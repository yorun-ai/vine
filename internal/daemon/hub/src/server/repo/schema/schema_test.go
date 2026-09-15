package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
)

func TestSchemaRepoSaveDomainSchemasOnce(t *testing.T) {
	repo := new(SchemaRepo)
	schema := testDomainSchema()

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})
	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	entry := repo.byHash[schema.Hash]
	require.NotNil(t, entry)
	assert.Same(t, schema, entry.Schema)
	assert.Len(t, repo.byHash, 1)
}

func TestSchemaRepoInstancesAreIndependent(t *testing.T) {
	writer := new(SchemaRepo)
	reader := new(SchemaRepo)
	schema := testDomainSchema()

	writer.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{schema})

	// Hub binds one repository per application, so readers inside an application
	// share state while separate instances stay independent.
	assert.Len(t, writer.ListAppConfigSchemas(), 1)
	assert.Empty(t, reader.ListDomainSchemaViews())
}

func TestSchemaRepoReleaseDomainSchemas(t *testing.T) {
	repo := new(SchemaRepo)
	oldSchema := testDomainSchema()
	newSchema := testDomainSchema()
	newSchema.Hash = "pkg-hash-2"

	repo.SaveDomainSchemas("demo.app", "instance-1", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("demo.app", "instance-2", []*skel.DomainSchema{newSchema})
	repo.ReleaseDomainSchemas("demo.app", "instance-2")

	_, ok := repo.byHash[newSchema.Hash]
	assert.False(t, ok)
	views := repo.ListDomainSchemaViews()
	require.Len(t, views, 1)
	assert.Same(t, oldSchema, views[0].DomainVersion.Schema)
	assert.True(t, views[0].DomainVersion.Main)
	assert.False(t, views[0].DomainVersion.MultiVersion)
}

func TestSchemaRepoGetWebSchemaTracksSelectedVersion(t *testing.T) {
	repo := new(SchemaRepo)
	oldSchema := testDomainSchema()
	newer := testDomainSchema()
	newer.Hash = "new-domain"
	newer.Webs[0].Hash = "new-web"
	newer.Webs[0].MountPath = "/new"
	name := oldSchema.Webs[0].SkelName
	got := repo.GetWebSchema(name)
	require.Nil(t, got)
	repo.SaveDomainSchemas("app", "old", []*skel.DomainSchema{oldSchema})
	repo.SaveDomainSchemas("app", "new", []*skel.DomainSchema{newer})
	got = repo.GetWebSchema(name)
	require.Same(t, newer.Webs[0], got)
	require.Equal(t, "/new", got.MountPath)
	repo.ReleaseDomainSchemas("app", "new")
	got = repo.GetWebSchema(name)
	require.Same(t, oldSchema.Webs[0], got)
	repo.ReleaseDomainSchemas("app", "old")
	got = repo.GetWebSchema(name)
	require.Nil(t, got)
}
