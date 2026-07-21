package schema

import (
	"cmp"
	"strings"

	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/core"
	"go.yorun.ai/vine/util/vslice"
)

func memoryBuildDomainSchemaVersions() []core.DomainSchemaVersion {
	versions := make([]core.DomainSchemaVersion, 0, len(memoryDomainSchemaByHash))
	for _, hashes := range memoryDomainSchemaHashesByDomain {
		entries := memoryActiveDomainSchemaEntries(hashes)
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
			versions = append(versions, core.DomainSchemaVersion{
				Schema:         entry.Schema,
				MainSchemaHash: mainEntry.Schema.Hash,
				Main:           entry == mainEntry,
				MultiVersion:   len(entries) > 1,
			})
		}
	}
	return vslice.SortBy(versions, func(a core.DomainSchemaVersion, b core.DomainSchemaVersion) bool {
		if a.Schema.Domain != b.Schema.Domain {
			return cmp.Compare(a.Schema.Domain, b.Schema.Domain) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.Schema.Hash, a.Schema.Hash) < 0
	})
}

func memoryBuildSchemaVersions[T any](
	domainVersions []core.DomainSchemaVersion,
	getRefs func(schema *skel.DomainSchema) []_MemorySchemaRef[T],
) []core.SchemaVersion[T] {
	states := memorySchemaVersionStates(domainVersions, getRefs, func(skelName string) bool {
		return !memoryIsVineSchemaRef(skelName)
	})
	ret := make([]core.SchemaVersion[T], 0)
	seen := map[string]struct{}{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Schema) {
			if memoryIsVineSchemaRef(ref.SkelName) {
				continue
			}
			key := memorySchemaVersionKey(ref.SkelName, ref.Hash)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			state := states[ref.SkelName]
			ret = append(ret, core.SchemaVersion[T]{
				Schema:           ref.Schema,
				Domain:           domainVersion.Schema.Domain,
				SkelName:         ref.SkelName,
				SchemaHash:       ref.Hash,
				MainSchemaHash:   state.DefaultHash,
				Main:             ref.Hash == state.DefaultHash,
				MultiVersion:     len(state.Hashes) > 1,
				DomainSchemaHash: domainVersion.Schema.Hash,
			})
		}
	}
	return vslice.SortBy(ret, func(a core.SchemaVersion[T], b core.SchemaVersion[T]) bool {
		if a.SkelName != b.SkelName {
			return cmp.Compare(a.SkelName, b.SkelName) < 0
		}
		if a.Main != b.Main {
			return a.Main
		}
		return cmp.Compare(b.SchemaHash, a.SchemaHash) < 0
	})
}

func memoryBuildDomainSchemaItemVersions[T any](
	domainVersion core.DomainSchemaVersion,
	states map[string]*_MemorySchemaVersionState,
	getRefs func(schema *skel.DomainSchema) []_MemorySchemaRef[T],
	include func(skelName string) bool,
) []core.SchemaVersion[T] {
	refs := getRefs(domainVersion.Schema)
	ret := make([]core.SchemaVersion[T], 0, len(refs))
	for _, ref := range refs {
		if !include(ref.SkelName) {
			continue
		}
		state := states[ref.SkelName]
		ret = append(ret, core.SchemaVersion[T]{
			Schema:           ref.Schema,
			Domain:           domainVersion.Schema.Domain,
			SkelName:         ref.SkelName,
			SchemaHash:       ref.Hash,
			MainSchemaHash:   state.DefaultHash,
			Main:             ref.Hash == state.DefaultHash,
			MultiVersion:     len(state.Hashes) > 1,
			DomainSchemaHash: domainVersion.Schema.Hash,
		})
	}
	return ret
}

func memorySchemaVersionStates[T any](
	domainVersions []core.DomainSchemaVersion,
	getRefs func(schema *skel.DomainSchema) []_MemorySchemaRef[T],
	include func(skelName string) bool,
) map[string]*_MemorySchemaVersionState {
	states := map[string]*_MemorySchemaVersionState{}
	for _, domainVersion := range domainVersions {
		for _, ref := range getRefs(domainVersion.Schema) {
			if !include(ref.SkelName) {
				continue
			}
			state := states[ref.SkelName]
			if state == nil {
				state = &_MemorySchemaVersionState{Hashes: map[string]struct{}{}}
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

func memorySchemaVersionKey(skelName string, hash string) string {
	return skelName + "\x00" + hash
}

func memoryIsVineSchemaRef(skelName string) bool {
	return strings.HasPrefix(skelName, "vine.")
}

func memoryIsVineHubSchemaRef(skelName string) bool {
	return strings.HasPrefix(skelName, "vine.hub.")
}
