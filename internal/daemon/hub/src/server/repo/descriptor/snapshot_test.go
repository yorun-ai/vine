package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
)

func TestDescriptorRepoListAppConfigDescriptors(t *testing.T) {
	repo := new(DescriptorRepo)
	descriptor := testDomainDescriptor()
	descriptor.Configs[0].Description = "Main config description"
	descriptor.Configs[0].Members = []*skeldesc.Member{
		{Name: "title", Description: "Page title"},
	}

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})

	descriptors := repo.ListAppConfigDescriptors()

	require.Len(t, descriptors, 1)
	assert.Same(t, descriptor.Configs[0], descriptors[0])
}

func TestDescriptorRepoListEnumDescriptors(t *testing.T) {
	repo := new(DescriptorRepo)
	descriptor := testDomainDescriptor()
	descriptor.Enums = []*skeldesc.Enum{{
		Name:     "UserStatus",
		SkelName: "demo.user.UserStatus",
		Items: []*skeldesc.EnumItem{
			{Name: "ACTIVE", Description: "启用"},
		},
	}}

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})

	descriptors := repo.ListEnumDescriptors()

	require.Len(t, descriptors, 1)
	assert.Same(t, descriptor.Enums[0], descriptors[0])
}

func TestDescriptorRepoListsLatestDomainDescriptorByDomain(t *testing.T) {
	repo := new(DescriptorRepo)
	oldDescriptor := testDomainDescriptor()
	newDescriptor := testDomainDescriptor()
	newDescriptor.Hash = "pkg-hash-2"
	newDescriptor.Actors = []*skeldesc.Actor{
		{
			Name:     "UserActor",
			SkelName: "demo.user.UserActor",
			Hash:     "actor-hash-2",
			Vias:     []skeldesc.ActorViaKind{skeldesc.ActorViaClient, skeldesc.ActorViaAgent},
		},
	}

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{oldDescriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-2", []*skeldesc.Domain{newDescriptor})

	actors := repo.ListActorDescriptors()

	require.Len(t, actors, 1)
	assert.Equal(t, "demo.user.UserActor", actors[0].SkelName)
	assert.Len(t, repo.byHash, 2)
}

func TestDescriptorRepoConfigTypeSnapshot(t *testing.T) {
	repo := new(DescriptorRepo)
	descriptor := testDomainDescriptor()
	descriptor.Data = []*skeldesc.Data{{SkelName: "demo.user.Settings"}}
	descriptor.Enums = []*skeldesc.Enum{{SkelName: "demo.user.Mode"}}
	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})
	configs, enums, data := repo.ListAppConfigTypeDescriptors()
	require.Equal(t, descriptor.Configs, configs)
	require.Equal(t, descriptor.Enums, enums)
	require.Equal(t, descriptor.Data, data)
}
