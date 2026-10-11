package skel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/skel/descriptor"
)

func TestValidateDescriptorAcceptsActorAudiences(t *testing.T) {
	domain := new(descriptor.Domain{
		Name: "demo", Generated: validGeneratedInfoForTest(),
		Actors: []*descriptor.Actor{{SkelName: "demo.Actor"}},
		Services: []*descriptor.Service{{
			Api: true, SkelName: "demo.Service", AuthMode: descriptor.AuthModeOptional,
			Audiences: []*descriptor.ActorAudience{{SkelName: "demo.Actor"}},
		}},
	})
	require.NoError(t, ValidateDescriptor(domain))
}
