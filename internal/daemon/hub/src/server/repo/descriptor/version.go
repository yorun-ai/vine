package descriptor

import (
	"cmp"
	"strings"

	skeldesc "go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

func (r *DescriptorRepo) buildDomainDescriptorVersions() []core.DomainDescriptorVersion {
	versions := make([]core.DomainDescriptorVersion, 0, len(r.byHash))
	for _, hashes := range r.hashesByDomain {
		entries := r.activeDomainDescriptorEntries(hashes)
		if len(entries) == 0 {
			continue
		}
		mainEntry := entries[0]
		for _, entry := range entries[1:] {
			if entry.Sequence > mainEntry.Sequence {
				mainEntry = entry
			}
		}
		for _, entry := range entries {
			versions = append(versions, core.DomainDescriptorVersion{
				Descriptor:         entry.Descriptor,
				MainDescriptorHash: mainEntry.Descriptor.Hash,
				Main:               entry == mainEntry,
				MultiVersion:       len(entries) > 1,
			})
		}
	}
	return vslice.SortBy(versions, func(a core.DomainDescriptorVersion, b core.DomainDescriptorVersion) bool {
		if a.Descriptor.Name != b.Descriptor.Name {
			return cmp.Compare(a.Descriptor.Name, b.Descriptor.Name) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.Descriptor.Hash, a.Descriptor.Hash) < 0
	})
}

func buildDescriptorVersions[T any](
	domainVersions []core.DomainDescriptorVersion,
	getRefs func(descriptor *skeldesc.Domain) []_DescriptorRef[T],
) []core.DescriptorVersion[T] {
	states := descriptorVersionStates(domainVersions, getRefs, func(skelName string) bool {
		return !isVineDescriptorRef(skelName)
	})
	ret := make([]core.DescriptorVersion[T], 0)
	seen := map[string]struct{}{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Descriptor) {
			if isVineDescriptorRef(ref.SkelName) {
				continue
			}
			key := descriptorVersionKey(ref.SkelName, ref.Hash)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			state := states[ref.SkelName]
			ret = append(ret, core.DescriptorVersion[T]{
				Descriptor:           ref.Descriptor,
				Domain:               domainVersion.Descriptor.Name,
				SkelName:             ref.SkelName,
				DescriptorHash:       ref.Hash,
				MainDescriptorHash:   state.DefaultHash,
				Main:                 ref.Hash == state.DefaultHash,
				MultiVersion:         len(state.Hashes) > 1,
				DomainDescriptorHash: domainVersion.Descriptor.Hash,
			})
		}
	}
	return vslice.SortBy(ret, func(a core.DescriptorVersion[T], b core.DescriptorVersion[T]) bool {
		if a.SkelName != b.SkelName {
			return cmp.Compare(a.SkelName, b.SkelName) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.DescriptorHash, a.DescriptorHash) < 0
	})
}

func buildDomainDescriptorItemVersions[T any](
	domainVersion core.DomainDescriptorVersion,
	states map[string]*_DescriptorVersionState,
	getRefs func(descriptor *skeldesc.Domain) []_DescriptorRef[T],
	include func(skelName string) bool,
) []core.DescriptorVersion[T] {
	refs := getRefs(domainVersion.Descriptor)
	ret := make([]core.DescriptorVersion[T], 0, len(refs))
	for _, ref := range refs {
		if !include(ref.SkelName) {
			continue
		}
		state := states[ref.SkelName]
		ret = append(ret, core.DescriptorVersion[T]{
			Descriptor:           ref.Descriptor,
			Domain:               domainVersion.Descriptor.Name,
			SkelName:             ref.SkelName,
			DescriptorHash:       ref.Hash,
			MainDescriptorHash:   state.DefaultHash,
			Main:                 ref.Hash == state.DefaultHash,
			MultiVersion:         len(state.Hashes) > 1,
			DomainDescriptorHash: domainVersion.Descriptor.Hash,
		})
	}
	return ret
}

func descriptorVersionStates[T any](
	domainVersions []core.DomainDescriptorVersion,
	getRefs func(descriptor *skeldesc.Domain) []_DescriptorRef[T],
	include func(skelName string) bool,
) map[string]*_DescriptorVersionState {
	states := map[string]*_DescriptorVersionState{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Descriptor) {
			if !include(ref.SkelName) {
				continue
			}
			state := states[ref.SkelName]
			if state == nil {
				state = &_DescriptorVersionState{
					Hashes: map[string]struct{}{},
				}
				states[ref.SkelName] = state
			}
			state.Hashes[ref.Hash] = struct{}{}
			if domainVersion.Main {
				state.MainDomainHash = ref.Hash
			}
		}
	}
	for _, state := range states {
		state.DefaultHash = state.MainDomainHash
		if state.DefaultHash == "" {
			for hash := range state.Hashes {
				if state.DefaultHash == "" || hash < state.DefaultHash {
					state.DefaultHash = hash
				}
			}
		}
	}
	return states
}

func descriptorVersionKey(skelName string, hash string) string {
	return skelName + "\x00" + hash
}

func isVineDescriptorRef(skelName string) bool {
	return strings.HasPrefix(skelName, "vine.")
}

func isVineHubDescriptorRef(skelName string) bool {
	return strings.HasPrefix(skelName, "vine.hub.")
}

func isVineHubDomain(domain string) bool {
	return domain == "vine.hub" || strings.HasPrefix(domain, "vine.hub.")
}
