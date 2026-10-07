package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func (s *Syncer) SyncDescriptors(domainViews []core.DomainDescriptorView) {
	s.descriptorMutex.Lock()
	defer s.descriptorMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	nextActorHashes := map[string]string{}
	nextResourceHashes := map[string]string{}
	nextServiceHashes := map[string]string{}
	nextWebHashes := map[string]string{}
	for _, view := range domainViews {
		for _, actorVersion := range view.Actors {
			if !actorVersion.Main {
				continue
			}
			actor := actorVersion.Descriptor
			if oldHash, ok := s.descriptorActorHashes[actor.SkelName]; !ok || oldHash != actor.Hash {
				batch.Set(watched.FormatDescriptorActorKey(actor.SkelName), vcode.MustMarshalJsonS(actor))
			}
			nextActorHashes[actor.SkelName] = actor.Hash
		}
		for _, resourceVersion := range view.Resources {
			if !resourceVersion.Main {
				continue
			}
			resource := resourceVersion.Descriptor
			if oldHash, ok := s.descriptorResourceHashes[resource.SkelName]; !ok || oldHash != resource.Hash {
				batch.Set(watched.FormatDescriptorResourceKey(resource.SkelName), vcode.MustMarshalJsonS(resource))
			}
			nextResourceHashes[resource.SkelName] = resource.Hash
		}
		for _, serviceVersion := range view.Services {
			if !serviceVersion.Main {
				continue
			}
			service := serviceVersion.Descriptor
			if !service.Api {
				continue
			}
			if oldHash, ok := s.descriptorServiceHashes[service.SkelName]; !ok || oldHash != service.Hash {
				batch.Set(watched.FormatDescriptorServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
			}
			nextServiceHashes[service.SkelName] = service.Hash
		}
		for _, webVersion := range view.Webs {
			if !webVersion.Main {
				continue
			}
			web := webVersion.Descriptor
			if oldHash, ok := s.descriptorWebHashes[web.SkelName]; !ok || oldHash != web.Hash {
				batch.Set(watched.FormatDescriptorWebKey(web.SkelName), vcode.MustMarshalJsonS(web))
			}
			nextWebHashes[web.SkelName] = web.Hash
		}
	}
	for actorSkelName := range s.descriptorActorHashes {
		if _, ok := nextActorHashes[actorSkelName]; !ok {
			batch.Delete(watched.FormatDescriptorActorKey(actorSkelName))
		}
	}
	for resourceSkelName := range s.descriptorResourceHashes {
		if _, ok := nextResourceHashes[resourceSkelName]; !ok {
			batch.Delete(watched.FormatDescriptorResourceKey(resourceSkelName))
		}
	}
	for serviceSkelName := range s.descriptorServiceHashes {
		if _, ok := nextServiceHashes[serviceSkelName]; !ok {
			batch.Delete(watched.FormatDescriptorServiceKey(serviceSkelName))
		}
	}
	for webSkelName := range s.descriptorWebHashes {
		if _, ok := nextWebHashes[webSkelName]; !ok {
			batch.Delete(watched.FormatDescriptorWebKey(webSkelName))
		}
	}
	batch.Notify()
	s.descriptorActorHashes = nextActorHashes
	s.descriptorResourceHashes = nextResourceHashes
	s.descriptorServiceHashes = nextServiceHashes
	s.descriptorWebHashes = nextWebHashes
}

// WriteDescriptors only writes descriptor keys and does not join the diff/delete lifecycle.
func (s *Syncer) WriteDescriptors(domainViews []core.DomainDescriptorView) {
	s.descriptorMutex.Lock()
	defer s.descriptorMutex.Unlock()

	for _, view := range domainViews {
		for _, actorVersion := range view.Actors {
			if !actorVersion.Main {
				continue
			}
			actor := actorVersion.Descriptor
			s.WatchServer.SetAndNotify(watched.FormatDescriptorActorKey(actor.SkelName), vcode.MustMarshalJsonS(actor))
		}
		for _, resourceVersion := range view.Resources {
			if !resourceVersion.Main {
				continue
			}
			resource := resourceVersion.Descriptor
			s.WatchServer.SetAndNotify(watched.FormatDescriptorResourceKey(resource.SkelName), vcode.MustMarshalJsonS(resource))
		}
		for _, serviceVersion := range view.Services {
			if !serviceVersion.Main {
				continue
			}
			service := serviceVersion.Descriptor
			if !service.Api {
				continue
			}
			s.WatchServer.SetAndNotify(watched.FormatDescriptorServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
		}
		for _, webVersion := range view.Webs {
			if !webVersion.Main {
				continue
			}
			web := webVersion.Descriptor
			s.WatchServer.SetAndNotify(watched.FormatDescriptorWebKey(web.SkelName), vcode.MustMarshalJsonS(web))
		}
	}
}
