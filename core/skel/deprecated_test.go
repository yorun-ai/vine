package skel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type legacyGeneratedSensitiveValue struct{}

func (legacyGeneratedSensitiveValue) SkelSensitive() {}

var _ Sensitive = legacyGeneratedSensitiveValue{}

type legacyGeneratedActor struct {
	ActorBase
}

func (legacyGeneratedActor) Name() string {
	return "LegacyActor"
}

func (legacyGeneratedActor) SkelName() string {
	return "legacy.LegacyActor"
}

func (legacyGeneratedActor) Vias() []ActorVia {
	return []ActorVia{ActorViaClient}
}

var _ Actor = legacyGeneratedActor{}

func TestRegisterDomainSchemaValidatesConvertedDescriptor(t *testing.T) {
	before := RegisteredDomainDescriptors()
	require.PanicsWithError(t, "convert legacy domain schema failed: nil legacy domain schema", func() {
		RegisterDomainSchema(nil)
	})
	require.Panics(t, func() {
		RegisterDomainSchema(&DomainSchema{
			Domain: t.Name(),
			Generated: &GeneratedInfo{
				CompilerVersion: "v99.0.0",
			},
			Services: []*ServiceSchema{{
				AuthMode: AuthModeOff,
			}},
		})
	})
	require.Equal(t, before, RegisteredDomainDescriptors())
}
