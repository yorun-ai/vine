package syncer

import (
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vine/util/vmap"
)

type _DescriptorValue struct {
	hash  string
	value any
}

func (s *Syncer) SyncDescriptors(domainViews []core.DomainDescriptorView) {
	s.descriptorMutex.Lock()
	defer s.descriptorMutex.Unlock()

	values := mainDescriptorValues(domainViews)
	batch := s.WatchServer.NotifyBatch()
	nextHashes := make(map[string]string, len(values))
	for _, key := range vmap.SortedKeys(values) {
		value := values[key]
		if oldHash, ok := s.descriptorHashes[key]; !ok || oldHash != value.hash {
			batch.Set(key, vcode.MustMarshalJsonS(value.value))
		}
		nextHashes[key] = value.hash
	}
	for _, key := range vmap.SortedKeys(s.descriptorHashes) {
		if _, ok := nextHashes[key]; !ok {
			batch.Delete(key)
		}
	}
	batch.Notify()
	s.descriptorHashes = nextHashes
}

// WriteDescriptors only writes descriptor keys and does not join the diff/delete lifecycle.
func (s *Syncer) WriteDescriptors(domainViews []core.DomainDescriptorView) {
	s.descriptorMutex.Lock()
	defer s.descriptorMutex.Unlock()

	values := mainDescriptorValues(domainViews)
	for _, key := range vmap.SortedKeys(values) {
		s.WatchServer.SetAndNotify(key, vcode.MustMarshalJsonS(values[key].value))
	}
}

func mainDescriptorValues(domainViews []core.DomainDescriptorView) map[string]_DescriptorValue {
	values := map[string]_DescriptorValue{}
	for _, view := range domainViews {
		if view.DomainVersion.Main {
			value := view.DomainVersion.Descriptor
			values[watched.FormatDescriptorDomainKey(value.Name)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Actors {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorActorKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Resources {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorResourceKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Services {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			if !value.Api {
				continue
			}
			values[watched.FormatDescriptorServiceKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Webs {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorWebKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Data {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorDataKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Enums {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorEnumKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Configs {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorConfigKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Events {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorEventKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}

		for _, version := range view.Tasks {
			if !version.Main {
				continue
			}
			value := version.Descriptor
			values[watched.FormatDescriptorTaskKey(value.SkelName)] = _DescriptorValue{hash: value.Hash, value: value}
		}
	}
	return values
}
