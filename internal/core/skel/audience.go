package skel

import (
	"slices"

	"go.yorun.ai/skel/descriptor"
)

// HasActorAudience matches an actor without interpreting legacy Via metadata.
func HasActorAudience(audiences []*descriptor.ActorAudience, actorSkelName string) bool {
	return slices.ContainsFunc(audiences, func(audience *descriptor.ActorAudience) bool {
		return audience.SkelName == actorSkelName
	})
}
