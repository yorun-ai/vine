package impl

import (
	"slices"

	skeltype "go.yorun.ai/skel/types"
	"go.yorun.ai/vine/internal/core/link/skeled"
)

// Link forwards opaque registration data. Hub owns decoding and validation.
func registrationDescriptors(registration skeled.AppRegistration) []skeltype.JSON {
	return append(slices.Clone(registration.DomainDescriptors), registration.DomainSchemas...)
}
