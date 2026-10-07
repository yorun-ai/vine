package descriptor

import (
	"cmp"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/util/vslice"
)

func actorDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Actor] {
	return descriptorRefs(descriptor.Actors, func(item *skeldesc.Actor) string { return item.SkelName }, func(item *skeldesc.Actor) string { return item.Hash })
}

func configDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Config] {
	return descriptorRefs(descriptor.Configs, func(item *skeldesc.Config) string { return item.SkelName }, func(item *skeldesc.Config) string { return item.Hash })
}

func dataDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Data] {
	refs := descriptorRefs(descriptor.Data, func(item *skeldesc.Data) string { return item.SkelName }, func(item *skeldesc.Data) string { return item.Hash })
	for _, actor := range descriptor.Actors {
		if actor.Auth != nil {
			refs = append(refs, _DescriptorRef[*skeldesc.Data]{
				SkelName:   actor.Auth.Credential.SkelName,
				Hash:       actor.Auth.Credential.Hash,
				Descriptor: actor.Auth.Credential,
			})
			refs = append(refs, _DescriptorRef[*skeldesc.Data]{
				SkelName:   actor.Auth.Info.SkelName,
				Hash:       actor.Auth.Info.Hash,
				Descriptor: actor.Auth.Info,
			})
		}
	}
	return refs
}

func enumDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Enum] {
	return descriptorRefs(descriptor.Enums, func(item *skeldesc.Enum) string { return item.SkelName }, func(item *skeldesc.Enum) string { return item.Hash })
}

func eventDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Event] {
	return descriptorRefs(descriptor.Events, func(item *skeldesc.Event) string { return item.SkelName }, func(item *skeldesc.Event) string { return item.Hash })
}

func resourceDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Resource] {
	return descriptorRefs(descriptor.Resources, func(item *skeldesc.Resource) string { return item.SkelName }, func(item *skeldesc.Resource) string { return item.Hash })
}

func serviceDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Service] {
	refs := descriptorRefs(descriptor.Services, func(item *skeldesc.Service) string { return item.SkelName }, func(item *skeldesc.Service) string { return item.Hash })
	for _, actor := range descriptor.Actors {
		if actor.Auth != nil {
			refs = append(refs, _DescriptorRef[*skeldesc.Service]{
				SkelName:   actor.Auth.Service.SkelName,
				Hash:       actor.Auth.Service.Hash,
				Descriptor: actor.Auth.Service,
			})
		}
		if actor.Permission != nil {
			refs = append(refs, _DescriptorRef[*skeldesc.Service]{
				SkelName:   actor.Permission.Service.SkelName,
				Hash:       actor.Permission.Service.Hash,
				Descriptor: actor.Permission.Service,
			})
		}
	}
	for _, resource := range descriptor.Resources {
		if resource.CheckService != nil {
			refs = append(refs, _DescriptorRef[*skeldesc.Service]{
				SkelName:   resource.CheckService.SkelName,
				Hash:       resource.CheckService.Hash,
				Descriptor: resource.CheckService,
			})
		}
	}
	return refs
}

func taskDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Task] {
	return descriptorRefs(descriptor.Tasks, func(item *skeldesc.Task) string { return item.SkelName }, func(item *skeldesc.Task) string { return item.Hash })
}

func webDescriptorRefs(descriptor *skeldesc.Domain) []_DescriptorRef[*skeldesc.Web] {
	return descriptorRefs(descriptor.Webs, func(item *skeldesc.Web) string { return item.SkelName }, func(item *skeldesc.Web) string { return item.Hash })
}

func descriptorRefs[T any](descriptors []T, skelNameOf func(T) string, hashOf func(T) string) []_DescriptorRef[T] {
	refs := make([]_DescriptorRef[T], 0, len(descriptors))
	for _, descriptor := range descriptors {
		refs = append(refs, _DescriptorRef[T]{
			SkelName:   skelNameOf(descriptor),
			Hash:       hashOf(descriptor),
			Descriptor: descriptor,
		})
	}
	return refs
}

func sortedDescriptorsBySkelName[T any](descriptors []T, skelNameOf func(T) string) []T {
	return vslice.SortBy(descriptors, func(a T, b T) bool {
		return cmp.Compare(skelNameOf(a), skelNameOf(b)) < 0
	})
}
