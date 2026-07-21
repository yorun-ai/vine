package impl

import (
	"cmp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

type _TestSchemaRef[T any] struct {
	SkelName string
	Hash     string
	Schema   T
}

type _TestSchemaVersionState struct {
	DefaultHash    string
	MainDomainHash string
	Hashes         map[string]struct{}
}

type _SkeletonServiceSchemaRepo struct {
	domainSchemas []*skel.DomainSchema
	versions      []core.DomainSchemaVersion
}

func (*_SkeletonServiceSchemaRepo) SaveDomainSchemas(string, string, []*skel.DomainSchema) {
}

func (*_SkeletonServiceSchemaRepo) SaveDomainSchemasJSON(string, string, []skel.JSON) {
}

func (*_SkeletonServiceSchemaRepo) ReleaseDomainSchemas(string, string) {}

func (r *_SkeletonServiceSchemaRepo) domainSchemaVersions() []core.DomainSchemaVersion {
	if r.versions != nil {
		return r.versions
	}
	versions := make([]core.DomainSchemaVersion, 0, len(r.domainSchemas))
	for _, schema := range r.domainSchemas {
		versions = append(versions, core.DomainSchemaVersion{Schema: schema, Main: true})
	}
	return versions
}

func (r *_SkeletonServiceSchemaRepo) ListDomainSchemaViews() []core.DomainSchemaView {
	versions := r.domainSchemaVersions()
	actorVersions := r.ListActorSchemaVersions()
	configVersions := r.ListConfigSchemaVersions()
	dataVersions := r.ListDataSchemaVersions()
	enumVersions := r.ListEnumSchemaVersions()
	eventVersions := r.ListEventSchemaVersions()
	resourceVersions := r.ListResourceSchemaVersions()
	serviceVersions := r.ListServiceSchemaVersions()
	taskVersions := r.ListTaskSchemaVersions()
	webVersions := r.ListWebSchemaVersions()
	views := make([]core.DomainSchemaView, 0, len(versions))
	for _, version := range versions {
		domainHash := version.Schema.Hash
		views = append(views, core.DomainSchemaView{
			DomainVersion: version,
			Actors:        testSchemaVersionsByDomainHash(actorVersions, domainHash),
			Configs:       testSchemaVersionsByDomainHash(configVersions, domainHash),
			Data:          testSchemaVersionsByDomainHash(dataVersions, domainHash),
			Enums:         testSchemaVersionsByDomainHash(enumVersions, domainHash),
			Events:        testSchemaVersionsByDomainHash(eventVersions, domainHash),
			Resources:     testSchemaVersionsByDomainHash(resourceVersions, domainHash),
			Services:      testSchemaVersionsByDomainHash(serviceVersions, domainHash),
			Tasks:         testSchemaVersionsByDomainHash(taskVersions, domainHash),
			Webs:          testSchemaVersionsByDomainHash(webVersions, domainHash),
		})
	}
	return views
}

func (r *_SkeletonServiceSchemaRepo) ListVineHubSchemaViews() []core.DomainSchemaView {
	return r.ListDomainSchemaViews()
}

func (r *_SkeletonServiceSchemaRepo) ListActorSchemaVersions() []core.SchemaVersion[*skel.ActorSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.ActorSchema] {
		refs := make([]_TestSchemaRef[*skel.ActorSchema], 0, len(schema.Actors))
		for _, item := range schema.Actors {
			refs = append(refs, _TestSchemaRef[*skel.ActorSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListConfigSchemaVersions() []core.SchemaVersion[*skel.ConfigSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.ConfigSchema] {
		refs := make([]_TestSchemaRef[*skel.ConfigSchema], 0, len(schema.Configs))
		for _, item := range schema.Configs {
			refs = append(refs, _TestSchemaRef[*skel.ConfigSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListDataSchemaVersions() []core.SchemaVersion[*skel.DataSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.DataSchema] {
		refs := make([]_TestSchemaRef[*skel.DataSchema], 0, len(schema.Data))
		for _, item := range schema.Data {
			refs = append(refs, _TestSchemaRef[*skel.DataSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		for _, actor := range schema.Actors {
			if actor.AuthCredential != nil {
				refs = append(refs, _TestSchemaRef[*skel.DataSchema]{SkelName: actor.AuthCredential.SkelName, Hash: actor.AuthCredential.Hash, Schema: actor.AuthCredential})
			}
			if actor.AuthInfo != nil {
				refs = append(refs, _TestSchemaRef[*skel.DataSchema]{SkelName: actor.AuthInfo.SkelName, Hash: actor.AuthInfo.Hash, Schema: actor.AuthInfo})
			}
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListEnumSchemaVersions() []core.SchemaVersion[*skel.EnumSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.EnumSchema] {
		refs := make([]_TestSchemaRef[*skel.EnumSchema], 0, len(schema.Enums))
		for _, item := range schema.Enums {
			refs = append(refs, _TestSchemaRef[*skel.EnumSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListEventSchemaVersions() []core.SchemaVersion[*skel.EventSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.EventSchema] {
		refs := make([]_TestSchemaRef[*skel.EventSchema], 0, len(schema.Events))
		for _, item := range schema.Events {
			refs = append(refs, _TestSchemaRef[*skel.EventSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListResourceSchemaVersions() []core.SchemaVersion[*skel.ResourceSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.ResourceSchema] {
		refs := make([]_TestSchemaRef[*skel.ResourceSchema], 0, len(schema.Resources))
		for _, item := range schema.Resources {
			refs = append(refs, _TestSchemaRef[*skel.ResourceSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListServiceSchemaVersions() []core.SchemaVersion[*skel.ServiceSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.ServiceSchema] {
		refs := make([]_TestSchemaRef[*skel.ServiceSchema], 0, len(schema.Services))
		for _, item := range schema.Services {
			refs = append(refs, _TestSchemaRef[*skel.ServiceSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		for _, actor := range schema.Actors {
			if actor.AuthService != nil {
				refs = append(refs, _TestSchemaRef[*skel.ServiceSchema]{SkelName: actor.AuthService.SkelName, Hash: actor.AuthService.Hash, Schema: actor.AuthService})
			}
			if actor.PermService != nil {
				refs = append(refs, _TestSchemaRef[*skel.ServiceSchema]{SkelName: actor.PermService.SkelName, Hash: actor.PermService.Hash, Schema: actor.PermService})
			}
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListTaskSchemaVersions() []core.SchemaVersion[*skel.TaskSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.TaskSchema] {
		refs := make([]_TestSchemaRef[*skel.TaskSchema], 0, len(schema.Tasks))
		for _, item := range schema.Tasks {
			refs = append(refs, _TestSchemaRef[*skel.TaskSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListWebSchemaVersions() []core.SchemaVersion[*skel.WebSchema] {
	return testSchemaVersions(r.domainSchemaVersions(), func(schema *skel.DomainSchema) []_TestSchemaRef[*skel.WebSchema] {
		refs := make([]_TestSchemaRef[*skel.WebSchema], 0, len(schema.Webs))
		for _, item := range schema.Webs {
			refs = append(refs, _TestSchemaRef[*skel.WebSchema]{SkelName: item.SkelName, Hash: item.Hash, Schema: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceSchemaRepo) ListActorSchemas() []*skel.ActorSchema {
	return nil
}

func (*_SkeletonServiceSchemaRepo) ListAppConfigSchemas() []*skel.ConfigSchema {
	return nil
}

func (r *_SkeletonServiceSchemaRepo) ListEnumSchemas() []*skel.EnumSchema {
	return nil
}

func (r *_SkeletonServiceSchemaRepo) ListServiceSchemas() []*skel.ServiceSchema {
	return nil
}

func (r *_SkeletonServiceSchemaRepo) ListWebSchemas() []*skel.WebSchema {
	return nil
}

func testSchemaVersions[T any](
	domainVersions []core.DomainSchemaVersion,
	getRefs func(schema *skel.DomainSchema) []_TestSchemaRef[T],
) []core.SchemaVersion[T] {
	states := map[string]*_TestSchemaVersionState{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Schema) {
			if strings.HasPrefix(ref.SkelName, "vine.") {
				continue
			}
			state := states[ref.SkelName]
			if state == nil {
				state = &_TestSchemaVersionState{Hashes: map[string]struct{}{}}
				states[ref.SkelName] = state
			}
			state.Hashes[ref.Hash] = struct{}{}
			if domainVersion.Main {
				state.MainDomainHash = ref.Hash
			}
		}
	}
	for _, state := range states {
		state.DefaultHash = state.MainDomainHash
		if state.DefaultHash == "" {
			for hash := range state.Hashes {
				if state.DefaultHash == "" || hash < state.DefaultHash {
					state.DefaultHash = hash
				}
			}
		}
	}

	ret := make([]core.SchemaVersion[T], 0)
	seen := map[string]struct{}{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Schema) {
			if strings.HasPrefix(ref.SkelName, "vine.") {
				continue
			}
			key := ref.SkelName + "\x00" + ref.Hash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			state := states[ref.SkelName]
			ret = append(ret, core.SchemaVersion[T]{
				Schema:           ref.Schema,
				Domain:           domainVersion.Schema.Domain,
				SkelName:         ref.SkelName,
				SchemaHash:       ref.Hash,
				MainSchemaHash:   state.DefaultHash,
				Main:             ref.Hash == state.DefaultHash,
				MultiVersion:     len(state.Hashes) > 1,
				DomainSchemaHash: domainVersion.Schema.Hash,
			})
		}
	}
	return vslice.SortBy(ret, func(a core.SchemaVersion[T], b core.SchemaVersion[T]) bool {
		if a.SkelName != b.SkelName {
			return cmp.Compare(a.SkelName, b.SkelName) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.SchemaHash, a.SchemaHash) < 0
	})
}

func testSchemaVersionsByDomainHash[T any](versions []core.SchemaVersion[T], domainHash string) []core.SchemaVersion[T] {
	ret := make([]core.SchemaVersion[T], 0)
	for _, version := range versions {
		if version.DomainSchemaHash == domainHash {
			ret = append(ret, version)
		}
	}
	return ret
}

func TestSkeletonServiceListServices(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Services: []*skel.ServiceSchema{
					{
						Name:     "AppConfigService",
						SkelName: "vine.hub.AppConfigService",
					},
					{
						Name:        "UserService",
						SkelName:    "demo.user.UserService",
						Description: "用户服务",
						Pub:         true,
						Require: &skel.PermRequire{
							Expr: &skel.PermExpr{
								Mode: skel.PermRequireModeCode,
								Code: "demo.user.User:read",
							},
						},
						Audiences: []*skel.ActorAudienceSchema{
							{Name: "UserActor", SkelName: "demo.user.UserActor"},
						},
						Methods: []*skel.MethodSchema{{
							Name:              "listUsers",
							SkelName:          "listUsers",
							Description:       "分页查询用户",
							InputDescription:  "分页参数",
							OutputDescription: "分页结果",
							Require: &skel.PermRequire{
								Expr: &skel.PermExpr{
									Mode: skel.PermRequireModeAny,
									Children: []*skel.PermExpr{
										{Mode: skel.PermRequireModeCode, Code: "demo.user.User:manage"},
										{
											Mode: skel.PermRequireModeCheck,
											Check: &skel.PermCheckInvocation{
												ResourceSkelName: "demo.user.User",
												ActionName:       "read",
												CheckName:        "byTenant",
												ServiceSkelName:  "demo.user.UserCheckService",
												MethodSkelName:   "checkByTenant",
												Arguments: []*skel.PermCheckArgument{{
													Name:     "tenantId",
													JsonPath: "params.tenantId",
													Type:     &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString},
												}},
											},
										},
									},
								},
							},
							Arguments: []*skel.MemberSchema{{
								Name:        "status",
								Description: "状态",
								Type: &skel.TypeSchema{
									Kind:     skel.TypeKindEnum,
									Name:     "UserStatus",
									SkelName: "demo.user.UserStatus",
									Nullable: true,
								},
							}},
							ResultType: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "Page",
								SkelName: "demo.user.Page",
								TypeArguments: []*skel.TypeSchema{{
									Kind:     skel.TypeKindData,
									Name:     "User",
									SkelName: "demo.user.User",
								}},
							},
						}},
					},
				},
			}},
		},
	}

	services := service.ListServices()

	require.Len(t, services, 1)
	assert.Equal(t, "demo.user", services[0].Domain)
	assert.Equal(t, "demo.user.UserService", services[0].SkelName)
	assert.True(t, services[0].Pub)
	require.NotNil(t, services[0].Require)
	assert.Equal(t, "demo.user.User:read", *services[0].Require.Code)
	require.Len(t, services[0].Actors, 1)
	assert.Equal(t, "demo.user.UserActor", services[0].Actors[0].SkelName)
	require.Len(t, services[0].Methods, 1)
	require.NotNil(t, services[0].Methods[0].Require)
	assert.Equal(t, "any", services[0].Methods[0].Require.Mode)
	require.Len(t, services[0].Methods[0].Require.Children, 2)
	assert.Equal(t, "demo.user.User:manage", *services[0].Methods[0].Require.Children[0].Code)
	require.NotNil(t, services[0].Methods[0].Require.Children[1].Check)
	assert.Equal(t, "checkByTenant", services[0].Methods[0].Require.Children[1].Check.MethodSkelName)
	assert.Equal(t, "params.tenantId", services[0].Methods[0].Require.Children[1].Check.Arguments[0].JsonPath)
	assert.Equal(t, "demo.user.Page<demo.user.User>", services[0].Methods[0].ResultType)
	require.Len(t, services[0].Methods[0].Arguments, 1)
	assert.Equal(t, "demo.user.UserStatus?", services[0].Methods[0].Arguments[0].Type)
}

func TestSkeletonServiceListResources(t *testing.T) {
	checkMethod := &skel.MethodSchema{
		Name:     "CheckByTenant",
		SkelName: "checkByTenant",
		Arguments: []*skel.MemberSchema{{
			Name: "tenantId",
			Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString},
		}},
	}
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Hash:   "domain-hash",
				Resources: []*skel.ResourceSchema{{
					Name:        "User",
					SkelName:    "demo.user.User",
					Hash:        "user-resource",
					Description: "用户资源",
					Checks: []*skel.ResourceCheckSchema{{
						Name:      "byTenant",
						Method:    checkMethod,
						Arguments: checkMethod.Arguments,
					}},
					Actions: []*skel.ResourceActionSchema{{
						Name:           "read",
						PermissionCode: "demo.user.User:read",
						Description:    "读取用户",
					}, {
						Name:           "update",
						PermissionCode: "demo.user.User:update",
						Checks: []*skel.ResourceCheckSchema{{
							Name:   "byTenant",
							Method: checkMethod,
							Arguments: []*skel.MemberSchema{{
								Name: "tenantId",
								Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString},
							}},
						}},
					}},
					CheckService: &skel.ServiceSchema{
						Name:     "UserCheckService",
						SkelName: "demo.user.UserCheckService",
						Hash:     "user-check-service",
						Methods:  []*skel.MethodSchema{checkMethod},
					},
				}},
			}},
		},
	}

	resources := service.ListResources()
	domains := service.ListDomains()

	require.Len(t, resources, 1)
	assert.Equal(t, "demo.user", resources[0].Domain)
	assert.Equal(t, "demo.user.User", resources[0].SkelName)
	assert.Equal(t, "用户资源", *resources[0].Description)
	require.Len(t, resources[0].Checks, 1)
	assert.Equal(t, "byTenant", resources[0].Checks[0].Name)
	assert.Equal(t, "checkByTenant", resources[0].Checks[0].MethodSkelName)
	assert.Equal(t, "string", resources[0].Checks[0].Arguments[0].Type)
	require.Len(t, resources[0].Actions, 2)
	assert.Equal(t, "demo.user.User:read", resources[0].Actions[0].PermissionCode)
	require.Len(t, resources[0].Actions[1].Checks, 1)
	require.NotNil(t, resources[0].CheckService)
	assert.Equal(t, "demo.user.UserCheckService", resources[0].CheckService.SkelName)
	require.Len(t, domains, 1)
	require.Len(t, domains[0].Resources, 1)
	assert.Equal(t, 1, domains[0].Total)
}

func TestSkeletonServiceFormatsExternalDomainTypesWithSkelName(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "booker",
				Hash:   "booker-hash",
				Data: []*skel.DataSchema{{
					Name:     "ReaderLoanContext",
					SkelName: "booker.ReaderLoanContext",
					Hash:     "reader-loan-context-hash",
					Members: []*skel.MemberSchema{
						{
							Name: "reader",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "UserSummary",
								SkelName: "user.UserSummary",
							},
						},
						{
							Name: "collaborators",
							Type: &skel.TypeSchema{
								Kind: skel.TypeKindList,
								Element: &skel.TypeSchema{
									Kind:     skel.TypeKindData,
									Name:     "UserSummary",
									SkelName: "user.UserSummary",
								},
							},
						},
						{
							Name: "books",
							Type: &skel.TypeSchema{
								Kind:     skel.TypeKindData,
								Name:     "Page",
								SkelName: "booker.Page",
								TypeArguments: []*skel.TypeSchema{{
									Kind:     skel.TypeKindData,
									Name:     "BookSummary",
									SkelName: "booker.BookSummary",
								}},
							},
						},
					},
				}},
			}},
		},
	}

	data := service.ListData()

	require.Len(t, data, 1)
	require.Len(t, data[0].Fields, 3)
	assert.Equal(t, "user.UserSummary", data[0].Fields[0].Type)
	assert.Equal(t, "list<user.UserSummary>", data[0].Fields[1].Type)
	assert.Equal(t, "booker.Page<booker.BookSummary>", data[0].Fields[2].Type)
}

func TestSkeletonServiceListActorsFiltersVineSkeletons(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Actors: []*skel.ActorSchema{
					{Name: "AdminActor", SkelName: "vine.hub.AdminActor"},
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				},
			}},
		},
	}

	actors := service.ListActors()

	require.Len(t, actors, 1)
	assert.Equal(t, "demo.user", actors[0].Domain)
	assert.Equal(t, "demo.user.UserActor", actors[0].SkelName)
}

func TestSkeletonServiceIncludesActorCredentialInfoAndAuthService(t *testing.T) {
	credential := &skel.DataSchema{
		Name:     "UserActorCredential",
		SkelName: "demo.user.UserActorCredential",
		Hash:     "credential-hash",
		Members: []*skel.MemberSchema{{
			Name: "token",
			Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString},
		}},
	}
	info := &skel.DataSchema{
		Name:     "UserActorInfo",
		SkelName: "demo.user.UserActorInfo",
		Hash:     "info-hash",
		Members: []*skel.MemberSchema{{
			Name: "userId",
			Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString},
		}},
	}
	authService := &skel.ServiceSchema{
		Name:     "UserActorAuthService",
		SkelName: "demo.user.UserActorAuthService",
		Hash:     "auth-service-hash",
		Methods: []*skel.MethodSchema{{
			Name:       "auth",
			SkelName:   "demo.user.UserActorAuthService.auth",
			ResultType: &skel.TypeSchema{Kind: skel.TypeKindData, Name: info.Name, SkelName: info.SkelName},
			Arguments: []*skel.MemberSchema{{
				Name: "credential",
				Type: &skel.TypeSchema{Kind: skel.TypeKindData, Name: credential.Name, SkelName: credential.SkelName},
			}},
		}},
	}
	permMethod := &skel.MethodSchema{
		Name:     "CheckAll",
		SkelName: "checkAll",
		Arguments: []*skel.MemberSchema{{
			Name: "codes",
			Type: &skel.TypeSchema{Kind: skel.TypeKindList, Element: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarString}},
		}},
	}
	permService := &skel.ServiceSchema{
		Name:     "UserActorPermissionService",
		SkelName: "demo.user.UserActorPermissionService",
		Hash:     "perm-service-hash",
		Methods:  []*skel.MethodSchema{permMethod},
	}
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Hash:   "domain-hash",
				Actors: []*skel.ActorSchema{{
					Name:           "UserActor",
					SkelName:       "demo.user.UserActor",
					Hash:           "actor-hash",
					AuthEnabled:    true,
					AuthCredential: credential,
					AuthInfo:       info,
					AuthService:    authService,
					PermEnabled:    true,
					PermService:    permService,
					PermMethod:     permMethod,
				}},
			}},
		},
	}

	actors := service.ListActors()
	data := service.ListData()
	services := service.ListServices()

	require.Len(t, actors, 1)
	assert.True(t, actors[0].AuthEnabled)
	require.NotNil(t, actors[0].Credential)
	assert.Equal(t, "demo.user.UserActorCredential", actors[0].Credential.SkelName)
	require.NotNil(t, actors[0].Info)
	assert.Equal(t, "demo.user.UserActorInfo", actors[0].Info.SkelName)
	require.NotNil(t, actors[0].AuthService)
	assert.Equal(t, "demo.user.UserActorAuthService", actors[0].AuthService.SkelName)
	assert.True(t, actors[0].PermEnabled)
	require.NotNil(t, actors[0].PermService)
	assert.Equal(t, "demo.user.UserActorPermissionService", actors[0].PermService.SkelName)
	require.NotNil(t, actors[0].PermMethod)
	assert.Equal(t, "checkAll", actors[0].PermMethod.SkelName)
	require.Len(t, data, 2)
	assert.Equal(t, "demo.user.UserActorCredential", data[0].SkelName)
	assert.Equal(t, "demo.user.UserActorInfo", data[1].SkelName)
	require.Len(t, services, 2)
	assert.Equal(t, "demo.user.UserActorAuthService", services[0].SkelName)
	assert.Equal(t, "demo.user.UserActorPermissionService", services[1].SkelName)
}

func TestSkeletonServiceListActorsIncludesAccessibleItems(t *testing.T) {
	mainSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-main",
		Actors: []*skel.ActorSchema{
			{Name: "UserActor", SkelName: "demo.user.UserActor", Hash: "actor-hash"},
		},
		Services: []*skel.ServiceSchema{
			{
				Name:     "MainService",
				SkelName: "demo.user.MainService",
				Hash:     "main-service",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				},
			},
		},
		Webs: []*skel.WebSchema{
			{
				Name:     "UserWeb",
				SkelName: "demo.user.UserWeb",
				Hash:     "user-web",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				},
			},
		},
		Events: []*skel.EventSchema{
			{
				Name:     "UserEvent",
				SkelName: "demo.user.UserEvent",
				Hash:     "user-event",
			},
		},
	}
	oldSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-old",
		Actors: []*skel.ActorSchema{
			{Name: "UserActor", SkelName: "demo.user.UserActor", Hash: "actor-hash"},
		},
		Services: []*skel.ServiceSchema{
			{
				Name:     "OldService",
				SkelName: "demo.user.OldService",
				Hash:     "old-service",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				},
			},
			{
				Name:     "OtherService",
				SkelName: "demo.user.OtherService",
				Hash:     "other-service",
				Audiences: []*skel.ActorAudienceSchema{
					{Name: "OtherActor", SkelName: "demo.user.OtherActor"},
				},
			},
		},
	}
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			versions: []core.DomainSchemaVersion{
				{Schema: oldSchema, MainSchemaHash: "domain-main", Main: false, MultiVersion: true},
				{Schema: mainSchema, MainSchemaHash: "domain-main", Main: true, MultiVersion: true},
			},
		},
	}

	actors := service.ListActors()

	require.Len(t, actors, 1)
	require.Len(t, actors[0].Services, 2)
	assert.Equal(t, "demo.user.MainService", actors[0].Services[0].SkelName)
	assert.Equal(t, "demo.user.OldService", actors[0].Services[1].SkelName)
	require.Len(t, actors[0].Webs, 1)
	assert.Equal(t, "demo.user.UserWeb", actors[0].Webs[0].SkelName)
}

func TestSkeletonServiceListActorsIncludesCrossDomainAccessibleItems(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{
				{
					Domain: "app",
					Hash:   "app-domain",
					Actors: []*skel.ActorSchema{
						{Name: "UserActor", SkelName: "app.UserActor", Hash: "actor-hash"},
					},
				},
				{
					Domain: "user",
					Hash:   "user-domain",
					Services: []*skel.ServiceSchema{
						{
							Name:     "UserService",
							SkelName: "user.UserService",
							Hash:     "user-service",
							Audiences: []*skel.ActorAudienceSchema{
								{Name: "UserActor", SkelName: "app.UserActor"},
							},
						},
					},
					Webs: []*skel.WebSchema{
						{
							Name:     "UserWeb",
							SkelName: "user.UserWeb",
							Hash:     "user-web",
							Audiences: []*skel.ActorAudienceSchema{
								{Name: "UserActor", SkelName: "app.UserActor"},
							},
						},
					},
					Events: []*skel.EventSchema{
						{
							Name:     "UserEvent",
							SkelName: "user.UserEvent",
							Hash:     "user-event",
						},
					},
				},
			},
		},
	}

	actors := service.ListActors()

	require.Len(t, actors, 1)
	assert.Equal(t, "app.UserActor", actors[0].SkelName)
	require.Len(t, actors[0].Services, 1)
	assert.Equal(t, "user.UserService", actors[0].Services[0].SkelName)
	require.Len(t, actors[0].Webs, 1)
	assert.Equal(t, "user.UserWeb", actors[0].Webs[0].SkelName)
}

func TestSkeletonServiceListConfigs(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Configs: []*skel.ConfigSchema{{
					Name:      "UserConfig",
					SkelName:  "demo.user.UserConfig",
					Pub:       true,
					Lifecycle: "ETERNAL",
					Members: []*skel.MemberSchema{{
						Name: "enabled",
						Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarBool},
					}},
				}},
			}},
		},
	}

	configs := service.ListConfigs()

	require.Len(t, configs, 1)
	assert.Equal(t, "demo.user", configs[0].Domain)
	assert.Equal(t, "demo.user.UserConfig", configs[0].SkelName)
	assert.True(t, configs[0].Pub)
	assert.Equal(t, "ETERNAL", configs[0].Lifecycle)
	require.Len(t, configs[0].Fields, 1)
	assert.Equal(t, "bool", configs[0].Fields[0].Type)
}

func TestSkeletonServiceListTasksAndEvents(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Tasks: []*skel.TaskSchema{{
					Name:     "SyncTask",
					SkelName: "demo.user.SyncTask",
					Triggers: []*skel.TriggerSchema{{
						Name:     "run",
						SkelName: "run",
						Arguments: []*skel.MemberSchema{{
							Name: "limit",
							Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarInt},
						}},
					}},
				}},
				Events: []*skel.EventSchema{{
					Name:     "UserCreatedEvent",
					SkelName: "demo.user.UserCreatedEvent",
					Pub:      true,
					Members: []*skel.MemberSchema{{
						Name: "userId",
						Type: &skel.TypeSchema{Kind: skel.TypeKindScalar, Scalar: skel.ScalarInt},
					}},
				}},
			}},
		},
	}

	tasks := service.ListTasks()
	events := service.ListEvents()

	require.Len(t, tasks, 1)
	assert.Equal(t, "demo.user", tasks[0].Domain)
	require.Len(t, tasks[0].Triggers, 1)
	assert.Equal(t, "int", tasks[0].Triggers[0].Arguments[0].Type)
	require.Len(t, events, 1)
	assert.Equal(t, "demo.user", events[0].Domain)
	assert.True(t, events[0].Pub)
	assert.Equal(t, "int", events[0].Fields[0].Type)
}

func TestSkeletonServiceListDataIncludesEnums(t *testing.T) {
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			domainSchemas: []*skel.DomainSchema{{
				Domain: "demo.user",
				Data: []*skel.DataSchema{
					{Name: "InternalData", SkelName: "vine.hub.InternalData"},
					{
						Name:           "Page",
						SkelName:       "demo.user.Page",
						Description:    "分页数据",
						TypeParameters: []string{"T"},
						Members: []*skel.MemberSchema{{
							Name: "items",
							Type: &skel.TypeSchema{Kind: skel.TypeKindList, Element: &skel.TypeSchema{Kind: skel.TypeKindTypeParameter, Name: "T"}},
						}},
					},
				},
				Enums: []*skel.EnumSchema{
					{Name: "InternalStatus", SkelName: "vine.hub.InternalStatus"},
					{
						Name:     "UserStatus",
						SkelName: "demo.user.UserStatus",
						Items: []*skel.EnumItemSchema{
							{Name: "ACTIVE", Description: "启用"},
						},
					},
				},
			}},
		},
	}

	data := service.ListData()

	require.Len(t, data, 2)
	assert.Equal(t, "demo.user", data[0].Domain)
	assert.Equal(t, "demo.user.Page", data[0].SkelName)
	assert.False(t, data[0].Enum)
	assert.Equal(t, []string{"T"}, data[0].TypeParameters)
	assert.Equal(t, "list<T>", data[0].Fields[0].Type)
	assert.Equal(t, "demo.user.UserStatus", data[1].SkelName)
	assert.Equal(t, "demo.user", data[1].Domain)
	assert.True(t, data[1].Enum)
	assert.Equal(t, "ACTIVE", data[1].EnumItems[0].Name)
}

func TestSkeletonServiceMergesItemVersionsOnServer(t *testing.T) {
	mainSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-main",
		Configs: []*skel.ConfigSchema{
			{Name: "StableConfig", SkelName: "demo.user.StableConfig", Hash: "stable-config"},
		},
		Services: []*skel.ServiceSchema{
			{Name: "StableService", SkelName: "demo.user.StableService", Hash: "stable-service"},
			{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-main"},
		},
		Data: []*skel.DataSchema{
			{Name: "StableData", SkelName: "demo.user.StableData", Hash: "stable-data"},
		},
	}
	oldSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-old",
		Configs: []*skel.ConfigSchema{
			{Name: "StableConfig", SkelName: "demo.user.StableConfig", Hash: "stable-config"},
		},
		Services: []*skel.ServiceSchema{
			{Name: "StableService", SkelName: "demo.user.StableService", Hash: "stable-service"},
			{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-old"},
			{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-b"},
		},
		Data: []*skel.DataSchema{
			{Name: "StableData", SkelName: "demo.user.StableData", Hash: "stable-data"},
		},
	}
	crossSchema := &skel.DomainSchema{
		Domain: "demo.user",
		Hash:   "domain-cross",
		Services: []*skel.ServiceSchema{
			{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-a"},
		},
	}
	service := &SkeletonServiceServerImpl{
		SchemaRepo: &_SkeletonServiceSchemaRepo{
			versions: []core.DomainSchemaVersion{
				{Schema: oldSchema, MainSchemaHash: "domain-main", Main: false, MultiVersion: true},
				{Schema: mainSchema, MainSchemaHash: "domain-main", Main: true, MultiVersion: true},
				{Schema: crossSchema, MainSchemaHash: "domain-main", Main: false, MultiVersion: true},
			},
		},
	}

	services := service.ListServices()
	configs := service.ListConfigs()
	data := service.ListData()
	domains := service.ListDomains()

	require.Len(t, services, 5)
	assert.Equal(t, "demo.user.ChangedService", services[0].SkelName)
	assert.True(t, services[0].IsMain)
	assert.True(t, services[0].IsMultiVersion)
	assert.Equal(t, "demo.user.ChangedService", services[1].SkelName)
	assert.False(t, services[1].IsMain)
	assert.True(t, services[1].IsMultiVersion)
	assert.Equal(t, "demo.user.RemovedService", services[2].SkelName)
	assert.True(t, services[2].IsMain)
	assert.Equal(t, "removed-service-a", services[2].SchemaHash)
	assert.Equal(t, "removed-service-a", services[2].MainSchemaHash)
	assert.True(t, services[2].IsMultiVersion)
	assert.Equal(t, "demo.user.RemovedService", services[3].SkelName)
	assert.False(t, services[3].IsMain)
	assert.Equal(t, "removed-service-b", services[3].SchemaHash)
	assert.Equal(t, "removed-service-a", services[3].MainSchemaHash)
	assert.True(t, services[3].IsMultiVersion)
	assert.Equal(t, "demo.user.StableService", services[4].SkelName)
	assert.True(t, services[4].IsMain)
	assert.False(t, services[4].IsMultiVersion)
	require.Len(t, configs, 1)
	assert.Equal(t, "stable-config", configs[0].SchemaHash)
	assert.False(t, configs[0].IsMultiVersion)
	require.Len(t, data, 1)
	assert.Equal(t, "stable-data", data[0].SchemaHash)
	assert.False(t, data[0].IsMultiVersion)
	require.Len(t, domains, 3)
	assert.Equal(t, "domain-main", domains[0].SchemaHash)
	assert.Equal(t, "domain-old", domains[1].SchemaHash)
	assert.False(t, domains[1].IsMain)
	require.Len(t, domains[1].Services, 3)
	require.Len(t, domains[1].Configs, 1)
	require.Len(t, domains[1].Data, 1)
	assert.Equal(t, "domain-cross", domains[2].SchemaHash)
	assert.False(t, domains[2].IsMain)
}
