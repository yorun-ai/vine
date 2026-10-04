package schema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
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

func TestSchemaRepoConvertsLegacyAuthModesAtBothInputs(t *testing.T) {
	for _, input := range []string{"json", "typed"} {
		t.Run(input, func(t *testing.T) {
			source := testDomainSchema()
			source.Services[0].AuthMode = skel.AuthModeAuth
			source.Services = append(source.Services, new(skel.ServiceSchema{SkelName: "demo.EmptyService"}), new(skel.ServiceSchema{SkelName: "demo.UnsetService", AuthMode: skel.AuthModeUnset}))
			source.Services[0].Methods = []*skel.MethodSchema{{SkelName: "get", Hash: "method", AuthMode: skel.AuthModeNoAuth}, {SkelName: "list", Hash: "list", AuthMode: skel.AuthModeUnset}}
			source.Webs[0].AuthMode = skel.AuthModeNoAuth
			source.Webs = append(source.Webs, new(skel.WebSchema{SkelName: "demo.EmptyWeb", Hash: "empty-web"}), new(skel.WebSchema{SkelName: "demo.UnsetWeb", Hash: "unset-web", AuthMode: skel.AuthModeUnset}))
			repo := new(SchemaRepo)
			if input == "json" {
				repo.SaveDomainSchemasJSON("demo", "instance", []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(source))})
			} else {
				repo.SaveDomainSchemas("demo", "instance", []*skel.DomainSchema{source})
			}
			registered := repo.byHash[source.Hash].Schema
			require.Equal(t, skel.AuthModeRequired, registered.Services[0].AuthMode)
			require.Equal(t, skel.AuthModeRequired, registered.Services[1].AuthMode)
			require.Equal(t, skel.AuthModeRequired, registered.Services[2].AuthMode)
			require.Equal(t, skel.AuthModeOptional, registered.Services[0].Methods[0].AuthMode)
			require.Equal(t, skel.AuthModeInherit, registered.Services[0].Methods[1].AuthMode)
			require.Equal(t, skel.AuthModeOff, registered.Webs[0].AuthMode)
			require.Equal(t, skel.AuthModeRequired, registered.Webs[1].AuthMode)
			require.Equal(t, skel.AuthModeRequired, registered.Webs[2].AuthMode)
			require.Equal(t, "empty-web", registered.Webs[1].Hash)
			require.Equal(t, "unset-web", registered.Webs[2].Hash)
			require.Equal(t, source.Hash, registered.Hash)
		})
	}
}

func TestSchemaRepoRejectsRpcOffBeforeReplacingOwner(t *testing.T) {
	for _, input := range []string{"json", "typed"} {
		for _, method := range []bool{false, true} {
			for _, sameHash := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/method=%t/sameHash=%t", input, method, sameHash), func(t *testing.T) {
					repo := new(SchemaRepo)
					original := testDomainSchema()
					repo.SaveDomainSchemas("demo", "instance", []*skel.DomainSchema{original})
					repo.SaveDomainSchemas("demo", "other", []*skel.DomainSchema{original})
					invalid := testDomainSchema()
					if !sameHash {
						invalid.Hash = "invalid"
					}
					service := invalid.Services[0]
					name := "Rpc service " + service.SkelName
					service.AuthMode = skel.AuthModeOff
					if method {
						service.AuthMode = skel.AuthModeRequired
						service.Methods = []*skel.MethodSchema{{SkelName: "get", AuthMode: skel.AuthModeOff}}
						name = "Rpc method " + service.SkelName + "/get"
					}
					require.PanicsWithError(t, "skel: "+name+" cannot use auth off; use auth optional for anonymous access", func() {
						if input == "json" {
							repo.SaveDomainSchemasJSON("demo", "instance", []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(invalid))})
						} else {
							repo.SaveDomainSchemas("demo", "instance", []*skel.DomainSchema{invalid})
						}
					})
					require.Same(t, original, repo.byHash[original.Hash].Schema)
					require.Len(t, repo.byHash, 1)
					views := repo.ListDomainSchemaViews()
					require.Len(t, views, 1)
					require.Same(t, original, views[0].DomainVersion.Schema)
					repo.ReleaseDomainSchemas("demo", "other")
					require.Len(t, repo.byHash, 1, "original ownership must survive rejection")
					repo.ReleaseDomainSchemas("demo", "instance")
					require.Empty(t, repo.byHash)
				})
			}
		}
	}
}

type registrationWriteSpy struct {
	core.RegistryRepo
	writes []string
}

func (r *registrationWriteSpy) SaveAppStatus(*core.AppStatus) { r.writes = append(r.writes, "app") }
func (r *registrationWriteSpy) SaveRpcServiceRegistration(*core.RpcServiceRegistration) {
	r.writes = append(r.writes, "rpc")
}
func (r *registrationWriteSpy) SaveWebRegistration(*core.WebRegistration) {
	r.writes = append(r.writes, "web")
}

func TestRejectedRegistrationDoesNotPublishEndpoints(t *testing.T) {
	for _, mode := range []skel.AuthMode{skel.AuthModeOff, skel.AuthModeOptional} {
		t.Run(string(mode), func(t *testing.T) {
			schemas := new(SchemaRepo)
			original := testDomainSchema()
			schemas.SaveDomainSchemas("demo", "instance", []*skel.DomainSchema{original})
			registry := new(registrationWriteSpy)
			target := core.RegistryCore{SchemaRepo: schemas, RegistryRepo: registry}
			invalid := testDomainSchema()
			invalid.Hash = "replacement"
			invalid.Services[0].AuthMode = mode
			reg := core.AppRegistration{Name: "demo", InstanceId: "instance",
				ServiceHandlers: []core.ServiceHandlerRegistration{{ServiceSkelName: invalid.Services[0].SkelName, Endpoint: "http://replacement"}},
				WebHandlers:     []core.WebHandlerRegistration{{WebSkelName: "demo.Web", Endpoint: "http://replacement"}},
				DomainSchemas:   []skel.JSON{skel.JSON(vcode.MustMarshalJsonS(invalid))}}
			if mode == skel.AuthModeOff {
				require.Panics(t, func() { target.Register(reg) })
				require.Empty(t, registry.writes)
				require.Same(t, original, schemas.byHash[original.Hash].Schema)
			} else {
				require.NotPanics(t, func() { target.Register(reg) })
				require.Equal(t, []string{"app", "rpc", "web"}, registry.writes)
				require.Contains(t, schemas.byHash, invalid.Hash)
			}
		})
	}
}
