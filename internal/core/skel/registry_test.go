package skel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/buildinfo"
)

func TestRegistrationRejectsInvalidDescriptorWithoutReplacingDomain(t *testing.T) {
	for _, mutate := range []func(*descriptor.Domain){
		func(d *descriptor.Domain) {
			d.Services[0].AuthMode = descriptor.AuthModeOff
		},
		func(d *descriptor.Domain) {
			d.Services[0].Methods[0].AuthMode = descriptor.AuthModeOff
		},
		func(d *descriptor.Domain) {
			d.Services[0].Methods[0].EffectiveAuthMode = descriptor.AuthModeOptional
		},
		func(d *descriptor.Domain) {
			d.Services[0].Methods[0].EffectiveRequire = &descriptor.PermissionRequire{
				Expression: &descriptor.PermissionExpression{
					Mode: descriptor.PermissionRequireModeCode,
					Code: "invented",
				},
			}
		},
		func(d *descriptor.Domain) {
			d.Services[0].Api = false
			d.Services[0].Audiences = []*descriptor.ActorAudience{{
				SkelName: "demo.User",
			}}
		},
		func(d *descriptor.Domain) {
			d.Services[0].Pub = true
		},
	} {
		registry := NewRegistry()
		original := &descriptor.Domain{
			Name:      "demo",
			Generated: validGeneratedInfoForTest(),
		}
		registry.RegisterDomainDescriptor(original)
		candidate := &descriptor.Domain{
			Name:      "demo",
			Full:      true,
			Generated: original.Generated,
			Services: []*descriptor.Service{{
				Name:     "Api",
				Api:      true,
				AuthMode: descriptor.AuthModeRequired,
				Methods: []*descriptor.Method{{
					Name:              "Get",
					SkelName:          "get",
					AuthMode:          descriptor.AuthModeInherit,
					EffectiveAuthMode: descriptor.AuthModeRequired,
				}},
			}},
		}
		mutate(candidate)
		require.Panics(t, func() {
			registry.RegisterDomainDescriptor(candidate)
		})
		require.Same(t, original, registry.RegisteredDomainDescriptors()[0])
	}
}

func TestRegisterDomainDescriptorPanicsOnDuplicatePartialDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "first-partial-hash",
		Generated: validGeneratedInfoForTest(),
		Services: []*descriptor.Service{
			{
				SkelName: "demo.user.PublicService",
				AuthMode: descriptor.AuthModeRequired,
			},
		},
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate partial domain to panic")
		}
	}()
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "next-partial-hash",
		Generated: validGeneratedInfoForTest(),
		Actors: []*descriptor.Actor{
			{SkelName: "demo.user.ClientActor"},
		},
	})
}

func TestRegisterDomainDescriptorFullOverridesRegisteredDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "pub-hash",
		Generated: validGeneratedInfoForTest(),
		Services: []*descriptor.Service{
			{
				SkelName: "demo.user.PublicService",
				AuthMode: descriptor.AuthModeRequired,
			},
		},
	})
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
		Actors: []*descriptor.Actor{
			{SkelName: "demo.user.ClientActor"},
		},
	})

	descriptors := registry.RegisteredDomainDescriptors()
	if len(descriptors) != 1 || descriptors[0].Hash != "full-hash" {
		t.Fatalf("expected full descriptor to override partial descriptor, got %+v", descriptors)
	}
}

func TestRegisteredDomainDescriptorsReturnsDomainSortedDescriptors(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "user-hash",
		Generated: validGeneratedInfoForTest(),
	})
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.booker",
		Hash:      "booker-hash",
		Generated: validGeneratedInfoForTest(),
	})
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.base",
		Hash:      "base-hash",
		Generated: validGeneratedInfoForTest(),
	})

	descriptors := registry.RegisteredDomainDescriptors()
	if len(descriptors) != 3 {
		t.Fatalf("unexpected registered descriptors count: %d", len(descriptors))
	}
	if descriptors[0].Name != "demo.base" || descriptors[1].Name != "demo.booker" || descriptors[2].Name != "demo.user" {
		t.Fatalf("unexpected descriptor order: %s, %s, %s", descriptors[0].Name, descriptors[1].Name, descriptors[2].Name)
	}
}

func TestRegisterDomainDescriptorPanicsOnDuplicateFullDomain(t *testing.T) {
	registry := NewRegistry()

	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
	})

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate full domain to panic")
		}
	}()
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name:      "demo.user",
		Hash:      "next-full-hash",
		Full:      true,
		Generated: validGeneratedInfoForTest(),
	})
}

func validGeneratedInfoForTest() *descriptor.GeneratedInfo {
	return &descriptor.GeneratedInfo{
		CompilerVersion: "v99.0.0",
	}
}

func TestRegisterDomainDescriptorSupportedCompilerVersions(t *testing.T) {
	for _, version := range []string{"v0.17.1", "v0.18.0", buildinfo.DevVersion} {
		t.Run(version, func(t *testing.T) {
			NewRegistry().RegisterDomainDescriptor(&descriptor.Domain{
				Name: "test",
				Generated: &descriptor.GeneratedInfo{
					CompilerVersion: version,
				},
			})
		})
	}
}

func TestRegisterDomainDescriptorPanicsOnLowCompilerVersion(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("expected low compiler version to panic")
		}
		if !strings.Contains(value.(error).Error(), "generated by skelc 0.17.0 is lower than Vine required skelc version") {
			t.Fatalf("unexpected panic: %v", value)
		}
	}()
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name: "demo.user",
		Hash: "full-hash",
		Generated: &descriptor.GeneratedInfo{
			CompilerVersion: "v0.17.0",
		},
	})
}

func TestRegisterDomainDescriptorPanicsOnMissingCompilerVersion(t *testing.T) {
	registry := NewRegistry()

	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("expected missing compiler version to panic")
		}
		if !strings.Contains(value.(error).Error(), "missing generated compiler version") {
			t.Fatalf("unexpected panic: %v", value)
		}
	}()
	registry.RegisterDomainDescriptor(&descriptor.Domain{
		Name: "demo.user",
		Hash: "full-hash",
	})
}
