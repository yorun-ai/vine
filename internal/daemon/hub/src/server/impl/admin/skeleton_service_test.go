package admin

import (
	"cmp"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	skeldesc "go.yorun.ai/skel/descriptor"
	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

func TestSkeletonApiPreservesExtensionMarkers(t *testing.T) {
	for _, ext := range []bool{false, true} {
		name := "public"
		if ext {
			name = "extension"
		}
		t.Run(name, func(t *testing.T) {
			service := &SkeletonApiServiceServerImpl{DescriptorRepo: &_SkeletonServiceDescriptorRepo{
				domainDescriptors: []*skeldesc.Domain{{Name: "demo.audit", Hash: "domain-hash",
					Services: []*skeldesc.Service{{Name: "AuditService", SkelName: "demo.audit.AuditService", Hash: "service-hash", Pub: true, Ext: ext, AuthMode: skeldesc.AuthModeRequired}},
					Events:   []*skeldesc.Event{{Name: "AuditRecordedEvent", SkelName: "demo.audit.AuditRecordedEvent", Hash: "event-hash", Pub: true, Ext: ext}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
				}},
			}}
			services := service.ListServices()
			events := service.ListEvents()
			domains := service.ListDomains()
			require.Len(t, services, 1)
			require.Len(t, events, 1)
			require.Len(t, domains, 1)
			require.Len(t, domains[0].Services, 1)
			require.Len(t, domains[0].Events, 1)
			assert.Equal(t, ext, services[0].Ext)
			assert.Equal(t, ext, events[0].Ext)
			assert.Equal(t, ext, domains[0].Services[0].Ext)
			assert.Equal(t, ext, domains[0].Events[0].Ext)
			for _, items := range []any{services, events} {
				encoded, err := json.Marshal(items)
				require.NoError(t, err)
				var decoded []map[string]any
				require.NoError(t, json.Unmarshal(encoded, &decoded))
				assert.Equal(t, ext, decoded[0]["ext"])
				assert.Equal(t, true, decoded[0]["pub"])
			}
		})
	}
}

type _TestDescriptorRef[T any] struct {
	SkelName   string
	Hash       string
	Descriptor T
}

type _TestDescriptorVersionState struct {
	DefaultHash    string
	MainDomainHash string
	Hashes         map[string]struct{}
}

type _SkeletonServiceDescriptorRepo struct {
	domainDescriptors []*skeldesc.Domain
	versions          []core.DomainDescriptorVersion
}

func (*_SkeletonServiceDescriptorRepo) SaveDomainDescriptors(string, string, []*skeldesc.Domain) {
}

func (*_SkeletonServiceDescriptorRepo) SaveDomainDescriptorsJSON(string, string, []skeltype.JSON) {
}

func (*_SkeletonServiceDescriptorRepo) ReleaseDomainDescriptors(string, string) {}

func (r *_SkeletonServiceDescriptorRepo) domainDescriptorVersions() []core.DomainDescriptorVersion {
	if r.versions != nil {
		return r.versions
	}
	versions := make([]core.DomainDescriptorVersion, 0, len(r.domainDescriptors))
	for _, descriptor := range r.domainDescriptors {
		versions = append(versions, core.DomainDescriptorVersion{Descriptor: descriptor, Main: true})
	}
	return versions
}

func (r *_SkeletonServiceDescriptorRepo) ListDomainDescriptorViews() []core.DomainDescriptorView {
	versions := r.domainDescriptorVersions()
	actorVersions := r.ListActorDescriptorVersions()
	configVersions := r.ListConfigDescriptorVersions()
	dataVersions := r.ListDataDescriptorVersions()
	enumVersions := r.ListEnumDescriptorVersions()
	eventVersions := r.ListEventDescriptorVersions()
	resourceVersions := r.ListResourceDescriptorVersions()
	serviceVersions := r.ListServiceDescriptorVersions()
	taskVersions := r.ListTaskDescriptorVersions()
	webVersions := r.ListWebDescriptorVersions()
	views := make([]core.DomainDescriptorView, 0, len(versions))
	for _, version := range versions {
		domainHash := version.Descriptor.Hash
		views = append(views, core.DomainDescriptorView{
			DomainVersion: version,
			Actors:        testDescriptorVersionsByDomainHash(actorVersions, domainHash),
			Configs:       testDescriptorVersionsByDomainHash(configVersions, domainHash),
			Data:          testDescriptorVersionsByDomainHash(dataVersions, domainHash),
			Enums:         testDescriptorVersionsByDomainHash(enumVersions, domainHash),
			Events:        testDescriptorVersionsByDomainHash(eventVersions, domainHash),
			Resources:     testDescriptorVersionsByDomainHash(resourceVersions, domainHash),
			Services:      testDescriptorVersionsByDomainHash(serviceVersions, domainHash),
			Tasks:         testDescriptorVersionsByDomainHash(taskVersions, domainHash),
			Webs:          testDescriptorVersionsByDomainHash(webVersions, domainHash),
		})
	}
	return views
}

func (r *_SkeletonServiceDescriptorRepo) ListVineHubDescriptorViews() []core.DomainDescriptorView {
	return r.ListDomainDescriptorViews()
}

func (r *_SkeletonServiceDescriptorRepo) ListActorDescriptorVersions() []core.DescriptorVersion[*skeldesc.Actor] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Actor] {
		refs := make([]_TestDescriptorRef[*skeldesc.Actor], 0, len(descriptor.Actors))
		for _, item := range descriptor.Actors {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Actor]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListConfigDescriptorVersions() []core.DescriptorVersion[*skeldesc.Config] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Config] {
		refs := make([]_TestDescriptorRef[*skeldesc.Config], 0, len(descriptor.Configs))
		for _, item := range descriptor.Configs {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Config]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListDataDescriptorVersions() []core.DescriptorVersion[*skeldesc.Data] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Data] {
		refs := make([]_TestDescriptorRef[*skeldesc.Data], 0, len(descriptor.Data))
		for _, item := range descriptor.Data {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Data]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		for _, actor := range descriptor.Actors {
			if actor.Auth != nil && actor.Auth.Credential != nil {
				refs = append(refs, _TestDescriptorRef[*skeldesc.Data]{SkelName: actor.Auth.Credential.SkelName, Hash: actor.Auth.Credential.Hash, Descriptor: actor.Auth.Credential})
			}
			if actor.Auth != nil && actor.Auth.Info != nil {
				refs = append(refs, _TestDescriptorRef[*skeldesc.Data]{SkelName: actor.Auth.Info.SkelName, Hash: actor.Auth.Info.Hash, Descriptor: actor.Auth.Info})
			}
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListEnumDescriptorVersions() []core.DescriptorVersion[*skeldesc.Enum] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Enum] {
		refs := make([]_TestDescriptorRef[*skeldesc.Enum], 0, len(descriptor.Enums))
		for _, item := range descriptor.Enums {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Enum]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListEventDescriptorVersions() []core.DescriptorVersion[*skeldesc.Event] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Event] {
		refs := make([]_TestDescriptorRef[*skeldesc.Event], 0, len(descriptor.Events))
		for _, item := range descriptor.Events {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Event]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListResourceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Resource] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Resource] {
		refs := make([]_TestDescriptorRef[*skeldesc.Resource], 0, len(descriptor.Resources))
		for _, item := range descriptor.Resources {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Resource]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListServiceDescriptorVersions() []core.DescriptorVersion[*skeldesc.Service] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Service] {
		refs := make([]_TestDescriptorRef[*skeldesc.Service], 0, len(descriptor.Services))
		for _, item := range descriptor.Services {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Service]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		for _, actor := range descriptor.Actors {
			if actor.Auth != nil && actor.Auth.Service != nil {
				refs = append(refs, _TestDescriptorRef[*skeldesc.Service]{SkelName: actor.Auth.Service.SkelName, Hash: actor.Auth.Service.Hash, Descriptor: actor.Auth.Service})
			}
			if actor.Permission != nil && actor.Permission.Service != nil {
				refs = append(refs, _TestDescriptorRef[*skeldesc.Service]{SkelName: actor.Permission.Service.SkelName, Hash: actor.Permission.Service.Hash, Descriptor: actor.Permission.Service})
			}
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListTaskDescriptorVersions() []core.DescriptorVersion[*skeldesc.Task] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Task] {
		refs := make([]_TestDescriptorRef[*skeldesc.Task], 0, len(descriptor.Tasks))
		for _, item := range descriptor.Tasks {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Task]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListWebDescriptorVersions() []core.DescriptorVersion[*skeldesc.Web] {
	return testDescriptorVersions(r.domainDescriptorVersions(), func(descriptor *skeldesc.Domain) []_TestDescriptorRef[*skeldesc.Web] {
		refs := make([]_TestDescriptorRef[*skeldesc.Web], 0, len(descriptor.Webs))
		for _, item := range descriptor.Webs {
			refs = append(refs, _TestDescriptorRef[*skeldesc.Web]{SkelName: item.SkelName, Hash: item.Hash, Descriptor: item})
		}
		return refs
	})
}

func (r *_SkeletonServiceDescriptorRepo) ListActorDescriptors() []*skeldesc.Actor {
	return nil
}

func (*_SkeletonServiceDescriptorRepo) ListAppConfigDescriptors() []*skeldesc.Config {
	return nil
}

func (r *_SkeletonServiceDescriptorRepo) ListEnumDescriptors() []*skeldesc.Enum {
	return nil
}

func (r *_SkeletonServiceDescriptorRepo) ListServiceDescriptors() []*skeldesc.Service {
	return nil
}

func (r *_SkeletonServiceDescriptorRepo) ListWebDescriptors() []*skeldesc.Web {
	return nil
}

func testDescriptorVersions[T any](
	domainVersions []core.DomainDescriptorVersion,
	getRefs func(descriptor *skeldesc.Domain) []_TestDescriptorRef[T],
) []core.DescriptorVersion[T] {
	states := map[string]*_TestDescriptorVersionState{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Descriptor) {
			if strings.HasPrefix(ref.SkelName, "vine.") {
				continue
			}
			state := states[ref.SkelName]
			if state == nil {
				state = &_TestDescriptorVersionState{Hashes: map[string]struct{}{}}
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

	ret := make([]core.DescriptorVersion[T], 0)
	seen := map[string]struct{}{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Descriptor) {
			if strings.HasPrefix(ref.SkelName, "vine.") {
				continue
			}
			key := ref.SkelName + "\x00" + ref.Hash
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			state := states[ref.SkelName]
			ret = append(ret, core.DescriptorVersion[T]{
				Descriptor:           ref.Descriptor,
				Domain:               domainVersion.Descriptor.Name,
				SkelName:             ref.SkelName,
				DescriptorHash:       ref.Hash,
				MainDescriptorHash:   state.DefaultHash,
				Main:                 ref.Hash == state.DefaultHash,
				MultiVersion:         len(state.Hashes) > 1,
				DomainDescriptorHash: domainVersion.Descriptor.Hash,
			})
		}
	}
	return vslice.SortBy(ret, func(a core.DescriptorVersion[T], b core.DescriptorVersion[T]) bool {
		if a.SkelName != b.SkelName {
			return cmp.Compare(a.SkelName, b.SkelName) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.DescriptorHash, a.DescriptorHash) < 0
	})
}

func testDescriptorVersionsByDomainHash[T any](versions []core.DescriptorVersion[T], domainHash string) []core.DescriptorVersion[T] {
	ret := make([]core.DescriptorVersion[T], 0)
	for _, version := range versions {
		if version.DomainDescriptorHash == domainHash {
			ret = append(ret, version)
		}
	}
	return ret
}

func TestSkeletonServiceListServices(t *testing.T) {
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Services: []*skeldesc.Service{
					{
						Name:     "AppConfigApiService",
						SkelName: "vine.hub.admin.AppConfigApiService", AuthMode: skeldesc.AuthModeRequired,
					},
					{
						Name:             "UserService",
						SkelName:         "demo.user.UserService",
						Description:      "用户服务",
						Deprecated:       true,
						DeprecatedReason: "Use UserServiceV2",
						Pub:              true,
						Require: &skeldesc.PermissionRequire{
							Expression: &skeldesc.PermissionExpression{
								Mode: skeldesc.PermissionRequireModeCode,
								Code: "demo.user.User:read",
							},
						},
						Audiences: []*skeldesc.ActorAudience{
							{Name: "UserActor", SkelName: "demo.user.UserActor"},
						},
						Methods: []*skeldesc.Method{{
							Name:               "listUsers",
							SkelName:           "listUsers",
							Description:        "分页查询用户",
							Deprecated:         true,
							DeprecatedReason:   "Use listUsersV2",
							InputDescription:   "分页参数",
							OutputDescription:  "分页结果",
							ArgumentsSensitive: true,
							ResultSensitive:    true,
							Require: &skeldesc.PermissionRequire{
								Expression: &skeldesc.PermissionExpression{
									Mode: skeldesc.PermissionRequireModeAny,
									Children: []*skeldesc.PermissionExpression{
										{Mode: skeldesc.PermissionRequireModeCode, Code: "demo.user.User:manage"},
										{
											Mode: skeldesc.PermissionRequireModeCheck,
											Check: &skeldesc.PermissionCheckInvocation{
												ResourceSkelName: "demo.user.User",
												ActionName:       "read",
												CheckName:        "byTenant",
												ServiceSkelName:  "demo.user.UserCheckService",
												MethodSkelName:   "checkByTenant",
												Arguments: []*skeldesc.PermissionCheckArgument{{
													Name:     "tenantId",
													JsonPath: "params.tenantId",
													Type:     &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString},
												}},
											},
										},
									},
								},
							},
							Arguments: []*skeldesc.Member{{
								Name:             "status",
								Description:      "状态",
								Deprecated:       true,
								DeprecatedReason: "Use statuses",
								Sensitive:        true,
								Type: &skeldesc.Type{
									Kind:     skeldesc.TypeKindEnum,
									Name:     "UserStatus",
									SkelName: "demo.user.UserStatus",
									Nullable: true,
								},
							}},
							ResultType: &skeldesc.Type{
								Kind:     skeldesc.TypeKindData,
								Name:     "Page",
								SkelName: "demo.user.Page",
								TypeArguments: []*skeldesc.Type{{
									Kind:     skeldesc.TypeKindData,
									Name:     "User",
									SkelName: "demo.user.User",
								}},
							}, AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired,
						}}, AuthMode: skeldesc.AuthModeRequired,
					},
				}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	services := service.ListServices()

	require.Len(t, services, 1)
	assert.Equal(t, "demo.user", services[0].Domain)
	assert.Equal(t, "demo.user.UserService", services[0].SkelName)
	assert.True(t, services[0].Deprecated)
	assert.Equal(t, "Use UserServiceV2", services[0].DeprecatedReason)
	assert.True(t, services[0].Pub)
	require.NotNil(t, services[0].Require)
	assert.Equal(t, "demo.user.User:read", services[0].Require.Code)
	require.Len(t, services[0].Actors, 1)
	assert.Equal(t, "demo.user.UserActor", services[0].Actors[0].SkelName)
	require.Len(t, services[0].Methods, 1)
	assert.True(t, services[0].Methods[0].Deprecated)
	assert.Equal(t, "Use listUsersV2", services[0].Methods[0].DeprecatedReason)
	require.NotNil(t, services[0].Methods[0].Require)
	assert.Equal(t, "any", services[0].Methods[0].Require.Mode)
	require.Len(t, services[0].Methods[0].Require.Children, 2)
	assert.Equal(t, "demo.user.User:manage", services[0].Methods[0].Require.Children[0].Code)
	require.NotNil(t, services[0].Methods[0].Require.Children[1].Check)
	assert.Equal(t, "checkByTenant", services[0].Methods[0].Require.Children[1].Check.MethodSkelName)
	assert.Equal(t, "params.tenantId", services[0].Methods[0].Require.Children[1].Check.Arguments[0].JsonPath)
	assert.Equal(t, "demo.user.Page<demo.user.User>", services[0].Methods[0].ResultType)
	assert.True(t, services[0].Methods[0].ArgumentsSensitive)
	assert.True(t, services[0].Methods[0].ResultSensitive)
	require.Len(t, services[0].Methods[0].Arguments, 1)
	assert.Equal(t, "demo.user.UserStatus?", services[0].Methods[0].Arguments[0].Type)
	assert.True(t, services[0].Methods[0].Arguments[0].Sensitive)
	assert.True(t, services[0].Methods[0].Arguments[0].Deprecated)
	assert.Equal(t, "Use statuses", services[0].Methods[0].Arguments[0].DeprecatedReason)
}

func TestSkeletonServiceListResources(t *testing.T) {
	checkMethod := &skeldesc.Method{
		Name:               "CheckByTenant",
		SkelName:           "checkByTenant",
		ArgumentsSensitive: true,
		Arguments: []*skeldesc.Member{{
			Name: "tenantId",
			Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString},
		}}, AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired,
	}
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Hash: "domain-hash",
				Resources: []*skeldesc.Resource{{
					Name:             "User",
					SkelName:         "demo.user.User",
					Hash:             "user-resource",
					Description:      "用户资源",
					Deprecated:       true,
					DeprecatedReason: "Use Account",
					Checks: []*skeldesc.ResourceCheck{{
						Name:             "byTenant",
						Deprecated:       true,
						DeprecatedReason: "Use byOrganization",
						MethodName:       checkMethod.Name,
					}},
					Actions: []*skeldesc.ResourceAction{{
						Name:             "read",
						PermissionCode:   "demo.user.User:read",
						Description:      "读取用户",
						Deprecated:       true,
						DeprecatedReason: "Use view",
					}, {
						Name:           "update",
						PermissionCode: "demo.user.User:update",
						Checks: []*skeldesc.ResourceCheck{{
							Name:       "byTenant",
							MethodName: checkMethod.Name,
						}},
					}},
					CheckService: &skeldesc.Service{
						Name:     "UserCheckService",
						SkelName: "demo.user.UserCheckService",
						Hash:     "user-check-service",
						Methods:  []*skeldesc.Method{checkMethod}, AuthMode: skeldesc.AuthModeRequired,
					},
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	resources := service.ListResources()
	domains := service.ListDomains()

	require.Len(t, resources, 1)
	assert.Equal(t, "demo.user", resources[0].Domain)
	assert.Equal(t, "demo.user.User", resources[0].SkelName)
	assert.Equal(t, "用户资源", resources[0].Description)
	assert.True(t, resources[0].Deprecated)
	assert.Equal(t, "Use Account", resources[0].DeprecatedReason)
	require.Len(t, resources[0].Checks, 1)
	assert.Equal(t, "byTenant", resources[0].Checks[0].Name)
	assert.Equal(t, "checkByTenant", resources[0].Checks[0].MethodSkelName)
	assert.Equal(t, "string", resources[0].Checks[0].Arguments[0].Type)
	assert.True(t, resources[0].Checks[0].ArgumentsSensitive)
	assert.True(t, resources[0].Checks[0].Deprecated)
	assert.Equal(t, "Use byOrganization", resources[0].Checks[0].DeprecatedReason)
	require.Len(t, resources[0].Actions, 2)
	assert.Equal(t, "demo.user.User:read", resources[0].Actions[0].PermissionCode)
	assert.True(t, resources[0].Actions[0].Deprecated)
	assert.Equal(t, "Use view", resources[0].Actions[0].DeprecatedReason)
	require.Len(t, resources[0].Actions[1].Checks, 1)
	require.NotNil(t, resources[0].CheckService)
	assert.Equal(t, "demo.user.UserCheckService", resources[0].CheckService.SkelName)
	require.Len(t, domains, 1)
	require.Len(t, domains[0].Resources, 1)
	assert.Equal(t, 1, domains[0].Total)
}

func TestSkeletonServiceFormatsExternalDomainTypesWithSkelName(t *testing.T) {
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "booker",
				Hash: "booker-hash",
				Data: []*skeldesc.Data{{
					Name:     "ReaderLoanContext",
					SkelName: "booker.ReaderLoanContext",
					Hash:     "reader-loan-context-hash",
					Members: []*skeldesc.Member{
						{
							Name: "reader",
							Type: &skeldesc.Type{
								Kind:     skeldesc.TypeKindData,
								Name:     "UserSummary",
								SkelName: "user.UserSummary",
							},
						},
						{
							Name: "collaborators",
							Type: &skeldesc.Type{
								Kind: skeldesc.TypeKindList,
								Element: &skeldesc.Type{
									Kind:     skeldesc.TypeKindData,
									Name:     "UserSummary",
									SkelName: "user.UserSummary",
								},
							},
						},
						{
							Name: "books",
							Type: &skeldesc.Type{
								Kind:     skeldesc.TypeKindData,
								Name:     "Page",
								SkelName: "booker.Page",
								TypeArguments: []*skeldesc.Type{{
									Kind:     skeldesc.TypeKindData,
									Name:     "BookSummary",
									SkelName: "booker.BookSummary",
								}},
							},
						},
					},
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
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
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Actors: []*skeldesc.Actor{
					{Name: "AdminActor", SkelName: "vine.hub.admin.AdminActor"},
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	actors := service.ListActors()

	require.Len(t, actors, 1)
	assert.Equal(t, "demo.user", actors[0].Domain)
	assert.Equal(t, "demo.user.UserActor", actors[0].SkelName)
}

func TestSkeletonServiceIncludesActorCredentialInfoAndAuthService(t *testing.T) {
	credential := &skeldesc.Data{
		Name:     "UserActorCredential",
		SkelName: "demo.user.UserActorCredential",
		Hash:     "credential-hash",
		Members: []*skeldesc.Member{{
			Name: "token",
			Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString},
		}},
	}
	info := &skeldesc.Data{
		Name:     "UserActorInfo",
		SkelName: "demo.user.UserActorInfo",
		Hash:     "info-hash",
		Members: []*skeldesc.Member{{
			Name: "userId",
			Type: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString},
		}},
	}
	authService := &skeldesc.Service{
		Name:     "UserActorAuthService",
		SkelName: "demo.user.UserActorAuthService",
		Hash:     "auth-service-hash",
		Methods: []*skeldesc.Method{{
			Name:       "auth",
			SkelName:   "demo.user.UserActorAuthService.auth",
			ResultType: &skeldesc.Type{Kind: skeldesc.TypeKindData, Name: info.Name, SkelName: info.SkelName},
			Arguments: []*skeldesc.Member{{
				Name: "credential",
				Type: &skeldesc.Type{Kind: skeldesc.TypeKindData, Name: credential.Name, SkelName: credential.SkelName},
			}}, AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired,
		}}, AuthMode: skeldesc.AuthModeRequired,
	}
	permMethod := &skeldesc.Method{
		Name:     "CheckAll",
		SkelName: "checkAll",
		Arguments: []*skeldesc.Member{{
			Name: "codes",
			Type: &skeldesc.Type{Kind: skeldesc.TypeKindList, Element: &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarString}},
		}}, AuthMode: skeldesc.AuthModeInherit, EffectiveAuthMode: skeldesc.AuthModeRequired,
	}
	permService := &skeldesc.Service{
		Name:     "UserActorPermissionService",
		SkelName: "demo.user.UserActorPermissionService",
		Hash:     "perm-service-hash",
		Methods:  []*skeldesc.Method{permMethod}, AuthMode: skeldesc.AuthModeRequired,
	}
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Hash: "domain-hash",
				Actors: []*skeldesc.Actor{{
					Name:     "UserActor",
					SkelName: "demo.user.UserActor",
					Hash:     "actor-hash", Auth: &skeldesc.ActorAuth{Credential: credential, Info: info, IdentifierField: "userId", Service: authService}, Permission: &skeldesc.ActorPermission{Service: permService, MethodName: permMethod.Name},
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	actors := service.ListActors()
	data := service.ListData()
	services := service.ListServices()

	require.Len(t, actors, 1)
	assert.True(t, actors[0].AuthEnabled)
	assert.Equal(t, "demo.user.UserActor", actors[0].SkelName)
	assert.Equal(t, "userId", actors[0].IdentifierField)
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
	mainDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "domain-main",
		Actors: []*skeldesc.Actor{
			{Name: "UserActor", SkelName: "demo.user.UserActor", Hash: "actor-hash"},
		},
		Services: []*skeldesc.Service{
			{
				Name:     "MainService",
				SkelName: "demo.user.MainService",
				Hash:     "main-service",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		},
		Webs: []*skeldesc.Web{
			{
				Name:     "UserWeb",
				SkelName: "demo.user.UserWeb",
				Hash:     "user-web",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		},
		Events: []*skeldesc.Event{
			{
				Name:     "UserEvent",
				SkelName: "demo.user.UserEvent",
				Hash:     "user-event",
			},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	oldDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "domain-old",
		Actors: []*skeldesc.Actor{
			{Name: "UserActor", SkelName: "demo.user.UserActor", Hash: "actor-hash"},
		},
		Services: []*skeldesc.Service{
			{
				Name:     "OldService",
				SkelName: "demo.user.OldService",
				Hash:     "old-service",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "UserActor", SkelName: "demo.user.UserActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
			{
				Name:     "OtherService",
				SkelName: "demo.user.OtherService",
				Hash:     "other-service",
				Audiences: []*skeldesc.ActorAudience{
					{Name: "OtherActor", SkelName: "demo.user.OtherActor"},
				}, AuthMode: skeldesc.AuthModeRequired,
			},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			versions: []core.DomainDescriptorVersion{
				{Descriptor: oldDescriptor, MainDescriptorHash: "domain-main", Main: false, MultiVersion: true},
				{Descriptor: mainDescriptor, MainDescriptorHash: "domain-main", Main: true, MultiVersion: true},
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
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{
				{
					Name: "app",
					Hash: "app-domain",
					Actors: []*skeldesc.Actor{
						{Name: "UserActor", SkelName: "app.UserActor", Hash: "actor-hash"},
					}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
				},
				{
					Name: "user",
					Hash: "user-domain",
					Services: []*skeldesc.Service{
						{
							Name:     "UserService",
							SkelName: "user.UserService",
							Hash:     "user-service",
							Audiences: []*skeldesc.ActorAudience{
								{Name: "UserActor", SkelName: "app.UserActor"},
							}, AuthMode: skeldesc.AuthModeRequired,
						},
					},
					Webs: []*skeldesc.Web{
						{
							Name:     "UserWeb",
							SkelName: "user.UserWeb",
							Hash:     "user-web",
							Audiences: []*skeldesc.ActorAudience{
								{Name: "UserActor", SkelName: "app.UserActor"},
							}, AuthMode: skeldesc.AuthModeRequired,
						},
					},
					Events: []*skeldesc.Event{
						{
							Name:     "UserEvent",
							SkelName: "user.UserEvent",
							Hash:     "user-event",
						},
					}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
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
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Configs: []*skeldesc.Config{{
					Name:             "UserConfig",
					SkelName:         "demo.user.UserConfig",
					Deprecated:       true,
					DeprecatedReason: "Use SiteConfig",
					Pub:              true,
					Sensitive:        true,
					Lifecycle:        "eternal",
					Members: []*skeldesc.Member{{
						Name:             "enabled",
						Deprecated:       true,
						DeprecatedReason: "Use active",
						Sensitive:        true,
						Type:             &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarBoolean},
					}},
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	configs := service.ListConfigs()

	require.Len(t, configs, 1)
	assert.Equal(t, "demo.user", configs[0].Domain)
	assert.Equal(t, "demo.user.UserConfig", configs[0].SkelName)
	assert.True(t, configs[0].Deprecated)
	assert.Equal(t, "Use SiteConfig", configs[0].DeprecatedReason)
	assert.True(t, configs[0].Pub)
	assert.True(t, configs[0].Sensitive)
	assert.Equal(t, "eternal", configs[0].Lifecycle)
	require.Len(t, configs[0].Fields, 1)
	assert.Equal(t, "bool", configs[0].Fields[0].Type)
	assert.True(t, configs[0].Fields[0].Sensitive)
	assert.True(t, configs[0].Fields[0].Deprecated)
	assert.Equal(t, "Use active", configs[0].Fields[0].DeprecatedReason)
}

func TestSkeletonServiceListTasksAndEvents(t *testing.T) {
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Tasks: []*skeldesc.Task{{
					Name:             "SyncTask",
					SkelName:         "demo.user.SyncTask",
					Deprecated:       true,
					DeprecatedReason: "Use ReconcileTask",
					Triggers: []*skeldesc.TaskTrigger{{
						Name:               "run",
						SkelName:           "run",
						Deprecated:         true,
						DeprecatedReason:   "Use scheduled",
						ArgumentsSensitive: true,
						Arguments: []*skeldesc.Member{{
							Name:      "limit",
							Sensitive: true,
							Type:      &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt},
						}},
					}},
				}},
				Events: []*skeldesc.Event{{
					Name:             "UserCreatedEvent",
					SkelName:         "demo.user.UserCreatedEvent",
					Deprecated:       true,
					DeprecatedReason: "Use AccountCreatedEvent",
					Pub:              true,
					Sensitive:        true,
					Members: []*skeldesc.Member{{
						Name:      "userId",
						Sensitive: true,
						Type:      &skeldesc.Type{Kind: skeldesc.TypeKindScalar, Scalar: skeldesc.ScalarInt},
					}},
				}}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	tasks := service.ListTasks()
	events := service.ListEvents()

	require.Len(t, tasks, 1)
	assert.True(t, tasks[0].Deprecated)
	assert.Equal(t, "Use ReconcileTask", tasks[0].DeprecatedReason)
	assert.Equal(t, "demo.user", tasks[0].Domain)
	require.Len(t, tasks[0].Triggers, 1)
	assert.True(t, tasks[0].Triggers[0].ArgumentsSensitive)
	assert.True(t, tasks[0].Triggers[0].Deprecated)
	assert.Equal(t, "Use scheduled", tasks[0].Triggers[0].DeprecatedReason)
	assert.Equal(t, "int", tasks[0].Triggers[0].Arguments[0].Type)
	assert.True(t, tasks[0].Triggers[0].Arguments[0].Sensitive)
	require.Len(t, events, 1)
	assert.Equal(t, "demo.user", events[0].Domain)
	assert.True(t, events[0].Pub)
	assert.True(t, events[0].Deprecated)
	assert.Equal(t, "Use AccountCreatedEvent", events[0].DeprecatedReason)
	assert.True(t, events[0].Sensitive)
	assert.Equal(t, "int", events[0].Fields[0].Type)
	assert.True(t, events[0].Fields[0].Sensitive)
}

func TestSkeletonServiceListDataIncludesEnums(t *testing.T) {
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			domainDescriptors: []*skeldesc.Domain{{
				Name: "demo.user",
				Data: []*skeldesc.Data{
					{Name: "InternalData", SkelName: "vine.hub.InternalData"},
					{
						Name:             "Page",
						SkelName:         "demo.user.Page",
						Description:      "分页数据",
						Deprecated:       true,
						DeprecatedReason: "Use CursorPage",
						Sensitive:        true,
						TypeParameters:   []string{"T"},
						Members: []*skeldesc.Member{{
							Name:      "items",
							Sensitive: true,
							Type:      &skeldesc.Type{Kind: skeldesc.TypeKindList, Element: &skeldesc.Type{Kind: skeldesc.TypeKindTypeParameter, Name: "T"}},
						}},
					},
				},
				Enums: []*skeldesc.Enum{
					{Name: "InternalStatus", SkelName: "vine.hub.InternalStatus"},
					{
						Name:             "UserStatus",
						SkelName:         "demo.user.UserStatus",
						Deprecated:       true,
						DeprecatedReason: "Use AccountStatus",
						Items: []*skeldesc.EnumItem{
							{Name: "ACTIVE", Description: "启用", Deprecated: true, DeprecatedReason: "Use ENABLED"},
						},
					},
				}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
			}},
		},
	}

	data := service.ListData()

	require.Len(t, data, 2)
	assert.Equal(t, "demo.user", data[0].Domain)
	assert.Equal(t, "demo.user.Page", data[0].SkelName)
	assert.False(t, data[0].Enum)
	assert.True(t, data[0].Deprecated)
	assert.Equal(t, "Use CursorPage", data[0].DeprecatedReason)
	assert.True(t, data[0].Sensitive)
	assert.Equal(t, []string{"T"}, data[0].TypeParameters)
	assert.Equal(t, "list<T>", data[0].Fields[0].Type)
	assert.True(t, data[0].Fields[0].Sensitive)
	assert.Equal(t, "demo.user.UserStatus", data[1].SkelName)
	assert.Equal(t, "demo.user", data[1].Domain)
	assert.True(t, data[1].Enum)
	assert.True(t, data[1].Deprecated)
	assert.Equal(t, "Use AccountStatus", data[1].DeprecatedReason)
	assert.Equal(t, "ACTIVE", data[1].EnumItems[0].Name)
	assert.True(t, data[1].EnumItems[0].Deprecated)
	assert.Equal(t, "Use ENABLED", data[1].EnumItems[0].DeprecatedReason)
}

func TestSkeletonServiceMergesItemVersionsOnServer(t *testing.T) {
	mainDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "domain-main",
		Configs: []*skeldesc.Config{
			{Name: "StableConfig", SkelName: "demo.user.StableConfig", Hash: "stable-config", Lifecycle: skeldesc.ConfigLifecycleEternal},
		},
		Services: []*skeldesc.Service{
			{Name: "StableService", SkelName: "demo.user.StableService", Hash: "stable-service", AuthMode: skeldesc.AuthModeRequired},
			{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-main", AuthMode: skeldesc.AuthModeRequired},
		},
		Data: []*skeldesc.Data{
			{Name: "StableData", SkelName: "demo.user.StableData", Hash: "stable-data"},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	oldDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "domain-old",
		Configs: []*skeldesc.Config{
			{Name: "StableConfig", SkelName: "demo.user.StableConfig", Hash: "stable-config", Lifecycle: skeldesc.ConfigLifecycleEternal},
		},
		Services: []*skeldesc.Service{
			{Name: "StableService", SkelName: "demo.user.StableService", Hash: "stable-service", AuthMode: skeldesc.AuthModeRequired},
			{Name: "ChangedService", SkelName: "demo.user.ChangedService", Hash: "changed-service-old", AuthMode: skeldesc.AuthModeRequired},
			{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-b", AuthMode: skeldesc.AuthModeRequired},
		},
		Data: []*skeldesc.Data{
			{Name: "StableData", SkelName: "demo.user.StableData", Hash: "stable-data"},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	crossDescriptor := &skeldesc.Domain{
		Name: "demo.user",
		Hash: "domain-cross",
		Services: []*skeldesc.Service{
			{Name: "RemovedService", SkelName: "demo.user.RemovedService", Hash: "removed-service-a", AuthMode: skeldesc.AuthModeRequired},
		}, Generated: &skeldesc.GeneratedInfo{CompilerVersion: "v99.0.0"},
	}
	service := &SkeletonApiServiceServerImpl{
		DescriptorRepo: &_SkeletonServiceDescriptorRepo{
			versions: []core.DomainDescriptorVersion{
				{Descriptor: oldDescriptor, MainDescriptorHash: "domain-main", Main: false, MultiVersion: true},
				{Descriptor: mainDescriptor, MainDescriptorHash: "domain-main", Main: true, MultiVersion: true},
				{Descriptor: crossDescriptor, MainDescriptorHash: "domain-main", Main: false, MultiVersion: true},
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
	assert.Equal(t, "removed-service-a", services[2].DescriptorHash)
	assert.Equal(t, "removed-service-a", services[2].MainDescriptorHash)
	assert.True(t, services[2].IsMultiVersion)
	assert.Equal(t, "demo.user.RemovedService", services[3].SkelName)
	assert.False(t, services[3].IsMain)
	assert.Equal(t, "removed-service-b", services[3].DescriptorHash)
	assert.Equal(t, "removed-service-a", services[3].MainDescriptorHash)
	assert.True(t, services[3].IsMultiVersion)
	assert.Equal(t, "demo.user.StableService", services[4].SkelName)
	assert.True(t, services[4].IsMain)
	assert.False(t, services[4].IsMultiVersion)
	require.Len(t, configs, 1)
	assert.Equal(t, "stable-config", configs[0].DescriptorHash)
	assert.False(t, configs[0].IsMultiVersion)
	require.Len(t, data, 1)
	assert.Equal(t, "stable-data", data[0].DescriptorHash)
	assert.False(t, data[0].IsMultiVersion)
	require.Len(t, domains, 3)
	assert.Equal(t, "domain-main", domains[0].DescriptorHash)
	assert.Equal(t, "domain-old", domains[1].DescriptorHash)
	assert.False(t, domains[1].IsMain)
	require.Len(t, domains[1].Services, 3)
	require.Len(t, domains[1].Configs, 1)
	require.Len(t, domains[1].Data, 1)
	assert.Equal(t, "domain-cross", domains[2].DescriptorHash)
	assert.False(t, domains[2].IsMain)
}

func TestSkeletonServiceApiFlag(t *testing.T) {
	for _, tc := range []struct {
		name     string
		api      bool
		pub      bool
		authMode skeldesc.AuthMode
	}{
		{name: "api", api: true},
		{name: "backend", pub: true},
		{name: "legacy", pub: true, authMode: skeldesc.AuthModeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := toServerSkeletonServiceItem(_SkeletonVersionFields{}, &skeldesc.Service{
				Api: tc.api, Pub: tc.pub, AuthMode: tc.authMode,
			})
			assert.Equal(t, tc.api, item.Api)
			assert.Equal(t, tc.pub, item.Pub)
		})
	}
}

func (r *_SkeletonServiceDescriptorRepo) GetWebDescriptor(skelName string) *skeldesc.Web {
	for _, descriptor := range r.ListWebDescriptors() {
		if descriptor.SkelName == skelName {
			return descriptor
		}
	}
	return nil
}

func (r *_SkeletonServiceDescriptorRepo) ListAppConfigTypeDescriptors() ([]*skeldesc.Config, []*skeldesc.Enum, []*skeldesc.Data) {
	return r.ListAppConfigDescriptors(), r.ListEnumDescriptors(), nil
}

func TestSkeletonWebAuthModes(t *testing.T) {
	for _, mode := range []skeldesc.AuthMode{"", skeldesc.AuthModeInherit, skeldesc.AuthModeRequired, skeldesc.AuthModeOptional, skeldesc.AuthModeAnonymous, skeldesc.AuthModeOff} {
		t.Run(string(mode), func(t *testing.T) {
			item := toServerSkeletonWebItem(_SkeletonVersionFields{}, new(skeldesc.Web{SkelName: "demo.Web", AuthMode: mode}))
			require.Equal(t, string(mode), item.AuthMode)
		})
	}
}
