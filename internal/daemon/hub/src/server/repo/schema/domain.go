package schema

import "go.yorun.ai/vine/internal/core/skel"

func memoryRetainDomainSchema(schema *skel.DomainSchema) *_MemoryDomainSchemaEntry {
	entry, ok := memoryDomainSchemaByHash[schema.Hash]
	if ok {
		return entry
	}

	memoryDomainSchemaSequence += 1
	entry = &_MemoryDomainSchemaEntry{
		Schema:   schema,
		Sequence: memoryDomainSchemaSequence,
		OwnerIDs: map[string]struct{}{},
	}
	memoryDomainSchemaByHash[schema.Hash] = entry
	if memoryDomainSchemaHashesByDomain[schema.Domain] == nil {
		memoryDomainSchemaHashesByDomain[schema.Domain] = map[string]struct{}{}
	}
	memoryDomainSchemaHashesByDomain[schema.Domain][schema.Hash] = struct{}{}
	return entry
}

func memoryRemoveDomainSchemaEntry(hash string, domain string) {
	delete(memoryDomainSchemaByHash, hash)
	delete(memoryDomainSchemaHashesByDomain[domain], hash)
	if len(memoryDomainSchemaHashesByDomain[domain]) == 0 {
		delete(memoryDomainSchemaHashesByDomain, domain)
	}
}

func memoryActiveDomainSchemaEntries(hashes map[string]struct{}) []*_MemoryDomainSchemaEntry {
	entries := make([]*_MemoryDomainSchemaEntry, 0, len(hashes))
	for hash := range hashes {
		entry := memoryDomainSchemaByHash[hash]
		if memoryDomainSchemaEntryActive(entry) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func memoryDomainSchemaEntryActive(entry *_MemoryDomainSchemaEntry) bool {
	return len(entry.OwnerIDs) > 0
}

func memoryOwnerKey(ownerName string, ownerId string) string {
	return ownerName + "\x00" + ownerId
}
