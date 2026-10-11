package skel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
)

func TestValidateDescriptorIgnoresViaMetadata(t *testing.T) {
	for _, via := range []descriptor.ActorViaKind{"", descriptor.ActorViaClient, descriptor.ActorViaAgent, descriptor.ActorViaOpenAPI, "future"} {
		t.Run(string(via), func(t *testing.T) {
			domain := new(descriptor.Domain{
				Name: "demo", Generated: validGeneratedInfoForTest(),
				Actors: []*descriptor.Actor{{SkelName: "demo.Actor", Vias: []descriptor.ActorViaKind{via}}},
				Services: []*descriptor.Service{{
					Api: true, SkelName: "demo.Service", AuthMode: descriptor.AuthModeOptional,
					Audiences: []*descriptor.ActorAudience{{SkelName: "demo.Actor", Via: via}},
				}},
			})
			require.NoError(t, ValidateDescriptor(domain))
			require.Equal(t, []descriptor.ActorViaKind{via}, domain.Actors[0].Vias)
			require.Equal(t, via, domain.Services[0].Audiences[0].Via)
		})
	}
}
