package schema

import "go.yorun.ai/vine/internal/core/skel"

func (r *SchemaRepo) retainDomainSchema(schema *skel.DomainSchema) *_DomainSchemaEntry {
	if entry, ok := r.byHash[schema.Hash]; ok {
		return entry
	}

	if r.byHash == nil {
		r.byHash = map[string]*_DomainSchemaEntry{}
	}
	if r.hashesByDomain == nil {
		r.hashesByDomain = map[string]map[string]struct{}{}
	}
	r.sequence++
	entry := &_DomainSchemaEntry{
		Schema:   schema,
		Sequence: r.sequence,
		OwnerIDs: map[string]struct{}{},
	}
	r.byHash[schema.Hash] = entry
	if r.hashesByDomain[schema.Domain] == nil {
		r.hashesByDomain[schema.Domain] = map[string]struct{}{}
	}
	r.hashesByDomain[schema.Domain][schema.Hash] = struct{}{}
	return entry
}

func (r *SchemaRepo) removeDomainSchemaEntry(hash string, domain string) {
	delete(r.byHash, hash)
	delete(r.hashesByDomain[domain], hash)
	if len(r.hashesByDomain[domain]) == 0 {
		delete(r.hashesByDomain, domain)
	}
}

func (r *SchemaRepo) activeDomainSchemaEntries(hashes map[string]struct{}) []*_DomainSchemaEntry {
	entries := make([]*_DomainSchemaEntry, 0, len(hashes))
	for hash := range hashes {
		entry := r.byHash[hash]
		if domainSchemaEntryActive(entry) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func domainSchemaEntryActive(entry *_DomainSchemaEntry) bool {
	return len(entry.OwnerIDs) > 0
}

func schemaOwnerKey(ownerName string, ownerId string) string {
	return ownerName + "\x00" + ownerId
}
