package descriptor

import (
	skeldesc "go.yorun.ai/skel/descriptor"
)

func (r *DescriptorRepo) retainDomainDescriptor(descriptor *skeldesc.Domain) *_DomainDescriptorEntry {
	if entry, ok := r.byHash[descriptor.Hash]; ok {
		return entry
	}

	if r.byHash == nil {
		r.byHash = map[string]*_DomainDescriptorEntry{}
	}
	if r.hashesByDomain == nil {
		r.hashesByDomain = map[string]map[string]struct{}{}
	}
	r.sequence++
	entry := &_DomainDescriptorEntry{
		Descriptor: descriptor,
		Sequence:   r.sequence,
		OwnerIDs:   map[string]struct{}{},
	}
	r.byHash[descriptor.Hash] = entry
	if r.hashesByDomain[descriptor.Name] == nil {
		r.hashesByDomain[descriptor.Name] = map[string]struct{}{}
	}
	r.hashesByDomain[descriptor.Name][descriptor.Hash] = struct{}{}
	return entry
}

func (r *DescriptorRepo) removeDomainDescriptorEntry(hash string, domain string) {
	delete(r.byHash, hash)
	delete(r.hashesByDomain[domain], hash)
	if len(r.hashesByDomain[domain]) == 0 {
		delete(r.hashesByDomain, domain)
	}
}

func (r *DescriptorRepo) activeDomainDescriptorEntries(hashes map[string]struct{}) []*_DomainDescriptorEntry {
	entries := make([]*_DomainDescriptorEntry, 0, len(hashes))
	for hash := range hashes {
		entry := r.byHash[hash]
		if domainDescriptorEntryActive(entry) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func domainDescriptorEntryActive(entry *_DomainDescriptorEntry) bool {
	return len(entry.OwnerIDs) > 0
}

func descriptorOwnerKey(ownerName string, ownerId string) string {
	return ownerName + "\x00" + ownerId
}
