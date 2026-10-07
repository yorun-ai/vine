package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
)

func TestDescriptorRepoSaveDomainDescriptors(t *testing.T) {
	repo := new(DescriptorRepo)
	oldDescriptor := testDomainDescriptor()
	newDescriptor := testDomainDescriptor()
	newDescriptor.Hash = "pkg-hash-2"

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{oldDescriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-2", []*skeldesc.Domain{newDescriptor})

	views := repo.ListDomainDescriptorViews()

	require.Len(t, views, 2)
	assert.Same(t, newDescriptor, views[0].DomainVersion.Descriptor)
	assert.True(t, views[0].DomainVersion.Main)
	assert.True(t, views[0].DomainVersion.MultiVersion)
	assert.Same(t, oldDescriptor, views[1].DomainVersion.Descriptor)
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

func TestDescriptorRepoListServiceDescriptorVersions(t *testing.T) {
	repo := new(DescriptorRepo)
	oldDescriptor := testDomainDescriptor()
	oldDescriptor.Hash = "domain-old"
	oldDescriptor.Services = []*skeldesc.Service{
		{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-old", AuthMode: skeldesc.AuthModeRequired},
		{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-b", AuthMode: skeldesc.AuthModeRequired},
	}
	mainDescriptor := testDomainDescriptor()
	mainDescriptor.Hash = "domain-main"
	mainDescriptor.Services = []*skeldesc.Service{
		{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-main", AuthMode: skeldesc.AuthModeRequired},
	}
	crossDescriptor := testDomainDescriptor()
	crossDescriptor.Hash = "domain-cross"
	crossDescriptor.Services = []*skeldesc.Service{
		{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-a", AuthMode: skeldesc.AuthModeRequired},
	}

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{oldDescriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-2", []*skeldesc.Domain{crossDescriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-3", []*skeldesc.Domain{mainDescriptor})

	versions := repo.ListServiceDescriptorVersions()

	require.Len(t, versions, 4)
	assert.Equal(t, "demo.user.ChangedService", versions[0].SkelName)
	assert.True(t, versions[0].Main)
	assert.Equal(t, "changed-service-main", versions[0].DescriptorHash)
	assert.Equal(t, "changed-service-main", versions[0].MainDescriptorHash)
	assert.True(t, versions[0].MultiVersion)
	assert.Equal(t, "demo.user.ChangedService", versions[1].SkelName)
	assert.False(t, versions[1].Main)
	assert.Equal(t, "changed-service-old", versions[1].DescriptorHash)
	assert.Equal(t, "changed-service-main", versions[1].MainDescriptorHash)
	assert.True(t, versions[1].MultiVersion)
	assert.Equal(t, "demo.user.RemovedService", versions[2].SkelName)
	assert.True(t, versions[2].Main)
	assert.Equal(t, "removed-service-a", versions[2].DescriptorHash)
	assert.Equal(t, "removed-service-a", versions[2].MainDescriptorHash)
	assert.True(t, versions[2].MultiVersion)
	assert.Equal(t, "demo.user.RemovedService", versions[3].SkelName)
	assert.False(t, versions[3].Main)
	assert.Equal(t, "removed-service-b", versions[3].DescriptorHash)
	assert.Equal(t, "removed-service-a", versions[3].MainDescriptorHash)
	assert.True(t, versions[3].MultiVersion)
}

func TestDescriptorRepoListsVineHubDescriptorViews(t *testing.T) {
	repo := new(DescriptorRepo)
	adminDescriptor := &skeldesc.Domain{
		Name: "vine.hub.admin",
		Hash: "hub-admin-domain-hash",
		Actors: []*skeldesc.Actor{{
			Name:     "AdminActor",
			SkelName: "vine.hub.admin.AdminActor",
			Hash:     "admin-actor-hash",
		}},
		Services: []*skeldesc.Service{{
			Name:     "SkeletonApiService",
			SkelName: "vine.hub.admin.SkeletonApiService",
			Hash:     "skeleton-service-hash", AuthMode: skeldesc.AuthModeRequired,
		}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	controlDescriptor := &skeldesc.Domain{
		Name: "vine.hub.control",
		Hash: "hub-control-domain-hash",
		Services: []*skeldesc.Service{{
			Name:     "InfoService",
			SkelName: "vine.hub.control.InfoService",
			Hash:     "info-service-hash", AuthMode: skeldesc.AuthModeRequired,
		}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}

	repo.SaveDomainDescriptors("vine.hub.inproc", "registered", []*skeldesc.Domain{controlDescriptor, adminDescriptor})

	assert.Empty(t, repo.ListActorDescriptorVersions())
	assert.Empty(t, repo.ListServiceDescriptorVersions())

	views := repo.ListVineHubDescriptorViews()
	require.Len(t, views, 2)
	foundControl := false
	foundAdmin := false
	for _, view := range views {
		switch view.DomainVersion.Descriptor.Name {
		case "vine.hub.control":
			foundControl = true
			require.Len(t, view.Services, 1)
			assert.Equal(t, "vine.hub.control.InfoService", view.Services[0].SkelName)
		case "vine.hub.admin":
			foundAdmin = true
			require.Len(t, view.Actors, 1)
			assert.Equal(t, "vine.hub.admin.AdminActor", view.Actors[0].SkelName)
			require.Len(t, view.Services, 1)
			assert.Equal(t, "vine.hub.admin.SkeletonApiService", view.Services[0].SkelName)
		}
	}
	assert.True(t, foundControl)
	assert.True(t, foundAdmin)
}
