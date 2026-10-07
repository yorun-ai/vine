package skel

import (
	"go.yorun.ai/skel/descriptor"
	internalskel "go.yorun.ai/vine/internal/core/skel"
)

// Actor is the Skel wire representation of an actor.
type Actor = internalskel.Actor

// ActorBase contains fields common to generated actor values.
type ActorBase = internalskel.ActorBase

// RegisterDomainDescriptor validates a contract and registers it in this process.
func RegisterDomainDescriptor(value *descriptor.Domain) {
	internalskel.RegisterDomainDescriptor(value)
}

// RegisteredDomainDescriptors returns contracts in stable domain-name order.
func RegisteredDomainDescriptors() []*descriptor.Domain {
	return internalskel.RegisteredDomainDescriptors()
}

// MinSkelcVersion returns the minimum skelc version required to generate code
// compatible with this Vine skel runtime.
func MinSkelcVersion() string {
	return internalskel.MinSkelcVersion()
}
