package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
)

func (s *Syncer) SyncSchemas(domainViews []core.DomainSchemaView) {
	s.schemaMutex.Lock()
	defer s.schemaMutex.Unlock()

	batch := s.WatchServer.NotifyBatch()
	nextActorHashes := map[string]string{}
	nextResourceHashes := map[string]string{}
	nextServiceHashes := map[string]string{}
	for _, view := range domainViews {
		for _, actorVersion := range view.Actors {
			if !actorVersion.Main {
				continue
			}
			actor := actorVersion.Schema
			if oldHash, ok := s.schemaActorHashes[actor.SkelName]; !ok || oldHash != actor.Hash {
				batch.Set(watched.FormatSchemaActorKey(actor.SkelName), vcode.MustMarshalJsonS(actor))
			}
			nextActorHashes[actor.SkelName] = actor.Hash
		}
		for _, resourceVersion := range view.Resources {
			if !resourceVersion.Main {
				continue
			}
			resource := resourceVersion.Schema
			if oldHash, ok := s.schemaResourceHashes[resource.SkelName]; !ok || oldHash != resource.Hash {
				batch.Set(watched.FormatSchemaResourceKey(resource.SkelName), vcode.MustMarshalJsonS(resource))
			}
			nextResourceHashes[resource.SkelName] = resource.Hash
		}
		for _, serviceVersion := range view.Services {
			if !serviceVersion.Main {
				continue
			}
			service := serviceVersion.Schema
			if !service.ClientApi() {
				continue
			}
			if oldHash, ok := s.schemaServiceHashes[service.SkelName]; !ok || oldHash != service.Hash {
				batch.Set(watched.FormatSchemaServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
			}
			nextServiceHashes[service.SkelName] = service.Hash
		}
	}
	for actorSkelName := range s.schemaActorHashes {
		if _, ok := nextActorHashes[actorSkelName]; !ok {
			batch.Delete(watched.FormatSchemaActorKey(actorSkelName))
		}
	}
	for resourceSkelName := range s.schemaResourceHashes {
		if _, ok := nextResourceHashes[resourceSkelName]; !ok {
			batch.Delete(watched.FormatSchemaResourceKey(resourceSkelName))
		}
	}
	for serviceSkelName := range s.schemaServiceHashes {
		if _, ok := nextServiceHashes[serviceSkelName]; !ok {
			batch.Delete(watched.FormatSchemaServiceKey(serviceSkelName))
		}
	}
	batch.Notify()
	s.schemaActorHashes = nextActorHashes
	s.schemaResourceHashes = nextResourceHashes
	s.schemaServiceHashes = nextServiceHashes
}

// WriteSchemas only writes schema keys and does not join the diff/delete lifecycle.
func (s *Syncer) WriteSchemas(domainViews []core.DomainSchemaView) {
	s.schemaMutex.Lock()
	defer s.schemaMutex.Unlock()

	for _, view := range domainViews {
		for _, actorVersion := range view.Actors {
			if !actorVersion.Main {
				continue
			}
			actor := actorVersion.Schema
			s.WatchServer.SetAndNotify(watched.FormatSchemaActorKey(actor.SkelName), vcode.MustMarshalJsonS(actor))
		}
		for _, resourceVersion := range view.Resources {
			if !resourceVersion.Main {
				continue
			}
			resource := resourceVersion.Schema
			s.WatchServer.SetAndNotify(watched.FormatSchemaResourceKey(resource.SkelName), vcode.MustMarshalJsonS(resource))
		}
		for _, serviceVersion := range view.Services {
			if !serviceVersion.Main {
				continue
			}
			service := serviceVersion.Schema
			if !service.ClientApi() {
				continue
			}
			s.WatchServer.SetAndNotify(watched.FormatSchemaServiceKey(service.SkelName), vcode.MustMarshalJsonS(service))
		}
	}
}
