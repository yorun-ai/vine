package skel

import (
	"go.yorun.ai/skel/descriptor"
	"go.yorun.ai/vine/util/vmap"
	"go.yorun.ai/vine/util/vpre"
)

// Registry stores validated descriptors. A full domain may replace its public projection.
type Registry struct {
	descriptorsByDomain map[string]*descriptor.Domain
}

// NewRegistry creates an isolated registry.
func NewRegistry() *Registry {
	return &Registry{
		descriptorsByDomain: map[string]*descriptor.Domain{},
	}
}

var defaultRegistry = NewRegistry()

// RegisterDomainDescriptor registers a descriptor in the process registry.
func RegisterDomainDescriptor(value *descriptor.Domain) {
	defaultRegistry.RegisterDomainDescriptor(value)
}

// RegisterDomainDescriptor validates before replacing a public projection.
func (r *Registry) RegisterDomainDescriptor(value *descriptor.Domain) {
	vpre.CheckNilError(ValidateDescriptor(value), "invalid domain descriptor")
	previous, exists := r.descriptorsByDomain[value.Name]
	vpre.CheckNot(exists && (previous.Full || !value.Full), "domain descriptor already registered: %s", value.Name)
	r.descriptorsByDomain[value.Name] = value
}

// RegisteredDomainDescriptors returns the process contracts sorted by domain.
func RegisteredDomainDescriptors() []*descriptor.Domain {
	return defaultRegistry.RegisteredDomainDescriptors()
}

// RegisteredDomainDescriptors returns this registry's contracts sorted by domain.
func (r *Registry) RegisteredDomainDescriptors() []*descriptor.Domain {
	return vmap.SortedValues(r.descriptorsByDomain)
}
