package control

import (
	"encoding/json/v2"
	"slices"

	"go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/core/skel/legacy"
	"go.yorun.ai/vine/internal/daemon/hub/api/skeled/control"
	"go.yorun.ai/vine/util/vpre"
)

// decodeRegisteredDescriptors is the remote registration boundary. The explicit
// domain/name discriminator also accepts old JSON forwarded opaquely by Link.
func decodeRegisteredDescriptors(reg control.AppRegistration) []*descriptor.Domain {
	values := append(slices.Clone(reg.DomainDescriptors), reg.DomainSchemas...)
	result := make([]*descriptor.Domain, 0, len(values))
	names := map[string]bool{}
	for _, raw := range values {
		var identity struct {
			Name   string `json:"name"`
			Domain string `json:"domain"`
		}
		vpre.MustNil(json.Unmarshal([]byte(raw), &identity))
		var value *descriptor.Domain
		switch {
		case identity.Name != "" && identity.Domain == "":
			vpre.MustNil(json.Unmarshal([]byte(raw), &value))
		case identity.Domain != "" && identity.Name == "":
			var previous legacy.DomainSchema
			vpre.MustNil(json.Unmarshal([]byte(raw), &previous))
			var err error
			value, err = legacy.Convert(&previous)
			vpre.MustNil(err)
		default:
			vpre.Panicf("registration requires exactly one domain name format")
		}
		vpre.MustNil(skel.ValidateDescriptor(value))
		vpre.CheckNot(names[value.Name], "duplicate registered domain %s", value.Name)
		names[value.Name] = true
		result = append(result, value)
	}
	return result
}
