package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
)

func TestDescriptorRepoSaveDomainDescriptorsOnce(t *testing.T) {
	repo := new(DescriptorRepo)
	descriptor := testDomainDescriptor()

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})

	entry := repo.byHash[descriptor.Hash]
	require.NotNil(t, entry)
	assert.Same(t, descriptor, entry.Descriptor)
	assert.Len(t, repo.byHash, 1)
}

func TestDescriptorRepoInstancesAreIndependent(t *testing.T) {
	writer := new(DescriptorRepo)
	reader := new(DescriptorRepo)
	descriptor := testDomainDescriptor()

	writer.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{descriptor})

	// Hub binds one repository per application, so readers inside an application
	// share state while separate instances stay independent.
	assert.Len(t, writer.ListAppConfigDescriptors(), 1)
	assert.Empty(t, reader.ListDomainDescriptorViews())
}

func TestDescriptorRepoReleaseDomainDescriptors(t *testing.T) {
	repo := new(DescriptorRepo)
	oldDescriptor := testDomainDescriptor()
	newDescriptor := testDomainDescriptor()
	newDescriptor.Hash = "pkg-hash-2"

	repo.SaveDomainDescriptors("demo.app", "instance-1", []*skeldesc.Domain{oldDescriptor})
	repo.SaveDomainDescriptors("demo.app", "instance-2", []*skeldesc.Domain{newDescriptor})
	repo.ReleaseDomainDescriptors("demo.app", "instance-2")

	_, ok := repo.byHash[newDescriptor.Hash]
	assert.False(t, ok)
	views := repo.ListDomainDescriptorViews()
	require.Len(t, views, 1)
	assert.Same(t, oldDescriptor, views[0].DomainVersion.Descriptor)
	assert.True(t, views[0].DomainVersion.Main)
	assert.False(t, views[0].DomainVersion.MultiVersion)
}

func TestDescriptorRepoGetWebDescriptorTracksSelectedVersion(t *testing.T) {
	repo := new(DescriptorRepo)
	oldDescriptor := testDomainDescriptor()
	newer := testDomainDescriptor()
	newer.Hash = "new-domain"
	newer.Webs[0].Hash = "new-web"
	newer.Webs[0].MountPath = "/new"
	name := oldDescriptor.Webs[0].SkelName
	got := repo.GetWebDescriptor(name)
	require.Nil(t, got)
	repo.SaveDomainDescriptors("app", "old", []*skeldesc.Domain{oldDescriptor})
	repo.SaveDomainDescriptors("app", "new", []*skeldesc.Domain{newer})
	got = repo.GetWebDescriptor(name)
	require.Same(t, newer.Webs[0], got)
	require.Equal(t, "/new", got.MountPath)
	repo.ReleaseDomainDescriptors("app", "new")
	got = repo.GetWebDescriptor(name)
	require.Same(t, oldDescriptor.Webs[0], got)
	repo.ReleaseDomainDescriptors("app", "old")
	got = repo.GetWebDescriptor(name)
	require.Nil(t, got)
}

func TestDescriptorRepoRejectsInvalidPolicyBeforeReplacingOwner(t *testing.T) {
	repo := new(DescriptorRepo)
	original := testDomainDescriptor()
	repo.SaveDomainDescriptors("demo", "instance", []*skeldesc.Domain{original})
	invalid := testDomainDescriptor()
	invalid.Hash = "replacement"
	invalid.Services[0].Methods = []*skeldesc.Method{{Name: "Get", SkelName: "get", AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeOptional}}
	require.Panics(t, func() { repo.SaveDomainDescriptors("demo", "instance", []*skeldesc.Domain{invalid}) })
	require.Same(t, original, repo.byHash[original.Hash].Descriptor)
	require.Len(t, repo.byHash, 1)
	repo.ReleaseDomainDescriptors("demo", "instance")
	require.Empty(t, repo.byHash)
}

type registrationWriteSpy struct {
	core.RegistryRepo
	writes []string
}

func (r *registrationWriteSpy) SaveAppStatus(*core.AppStatus) {
	r.writes = append(r.writes, "app")
}

func (r *registrationWriteSpy) SaveRpcServiceRegistration(*core.RpcServiceRegistration) {
	r.writes = append(r.writes, "rpc")
}
func (r *registrationWriteSpy) SaveWebRegistration(*core.WebRegistration) {
	r.writes = append(r.writes, "web")
}

func TestRejectedRegistrationDoesNotPublishEndpoints(t *testing.T) {
	for _, mode := range []skeldesc.AuthMode{skeldesc.AuthModeOff, skeldesc.AuthModeOptional} {
		t.Run(string(mode), func(t *testing.T) {
			descriptors := new(DescriptorRepo)
			original := testDomainDescriptor()
			descriptors.SaveDomainDescriptors("demo", "instance", []*skeldesc.Domain{original})
			registry := new(registrationWriteSpy)
			target := core.RegistryCore{DescriptorRepo: descriptors, RegistryRepo: registry}
			invalid := testDomainDescriptor()
			invalid.Hash = "replacement"
			invalid.Services[0].AuthMode = mode
			reg := core.AppRegistration{Name: "demo", InstanceId: "instance",
				ServiceHandlers:   []core.ServiceHandlerRegistration{{ServiceSkelName: invalid.Services[0].SkelName, Endpoint: "http://replacement"}},
				WebHandlers:       []core.WebHandlerRegistration{{WebSkelName: "demo.Web", Endpoint: "http://replacement"}},
				DomainDescriptors: []*skeldesc.Domain{invalid}}
			if mode == skeldesc.AuthModeOff {
				require.Panics(t, func() { target.Register(reg) })
				require.Empty(t, registry.writes)
				require.Same(t, original, descriptors.byHash[original.Hash].Descriptor)
			} else {
				require.NotPanics(t, func() { target.Register(reg) })
				require.Equal(t, []string{"app", "rpc", "web"}, registry.writes)
				require.Contains(t, descriptors.byHash, invalid.Hash)
			}
		})
	}
}
