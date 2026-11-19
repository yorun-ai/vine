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
func decodeRegisteredDescriptors(reg control.AppRegistration) ([]*descriptor.Domain, _LegacyRegistrationHashes) {
	values := append(slices.Clone(reg.DomainDescriptors), reg.DomainSchemas...)
	result := make([]*descriptor.Domain, 0, len(values))
	var hashes _LegacyRegistrationHashes
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
			hashes.collect(value)
		default:
			vpre.Panicf("registration requires exactly one domain name format")
		}
		vpre.MustNil(skel.ValidateDescriptor(value))
		vpre.CheckNot(names[value.Name], "duplicate registered domain %s", value.Name)
		names[value.Name] = true
		result = append(result, value)
	}
	return result, hashes
}

// v0.27 registrations used schemaHash, which can be dropped on the way through
// Link. Recover it only from definitions supplied by the same App instance.
type _LegacyRegistrationHashes struct {
	services _LegacyHashes
	webs     _LegacyHashes
	events   _LegacyHashes
	tasks    _LegacyHashes
}

type _LegacyHashes map[string]string

func (h *_LegacyHashes) add(name string, hash string) {
	if name == "" || hash == "" {
		return
	}
	if *h == nil {
		*h = make(_LegacyHashes)
	}
	if previous, seen := (*h)[name]; seen && previous != hash {
		// An ambiguous definition stays unresolved for this registration.
		(*h)[name] = ""
		return
	}
	(*h)[name] = hash
}

func (h *_LegacyRegistrationHashes) collect(domain *descriptor.Domain) {
	addService := func(service *descriptor.Service) {
		if service != nil {
			h.services.add(service.SkelName, service.Hash)
		}
	}
	for _, service := range domain.Services {
		addService(service)
	}
	for _, actor := range domain.Actors {
		if actor == nil {
			continue
		}
		if actor.Auth != nil {
			addService(actor.Auth.Service)
		}
		if actor.Permission != nil {
			addService(actor.Permission.Service)
		}
	}
	for _, resource := range domain.Resources {
		if resource != nil {
			addService(resource.CheckService)
		}
	}
	for _, web := range domain.Webs {
		if web != nil {
			h.webs.add(web.SkelName, web.Hash)
		}
	}
	for _, event := range domain.Events {
		if event != nil {
			h.events.add(event.SkelName, event.Hash)
		}
	}
	for _, task := range domain.Tasks {
		if task != nil {
			h.tasks.add(task.SkelName, task.Hash)
		}
	}
}
