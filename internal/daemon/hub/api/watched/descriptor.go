package watched

import (
	"fmt"

	skeldesc "go.yorun.ai/skel/descriptor"
)

const (
	descriptorActorPrefix    = "descriptor:actor"
	descriptorActorKeyFormat = descriptorActorPrefix + ":%s"

	descriptorServicePrefix    = "descriptor:service"
	descriptorServiceKeyFormat = descriptorServicePrefix + ":%s"

	descriptorResourcePrefix    = "descriptor:resource"
	descriptorResourceKeyFormat = descriptorResourcePrefix + ":%s"
)

type DescriptorWeb = skeldesc.Web

type DescriptorActor = skeldesc.Actor
type DescriptorService = skeldesc.Service
type DescriptorResource = skeldesc.Resource

func FormatDescriptorActorKey(actorSkelName string) string {
	return fmt.Sprintf(descriptorActorKeyFormat, actorSkelName)
}

func FormatDescriptorActorPrefix() string {
	return descriptorActorPrefix
}

func FormatDescriptorServiceKey(serviceSkelName string) string {
	return fmt.Sprintf(descriptorServiceKeyFormat, serviceSkelName)
}

func FormatDescriptorServicePrefix() string {
	return descriptorServicePrefix
}

func FormatDescriptorResourceKey(resourceSkelName string) string {
	return fmt.Sprintf(descriptorResourceKeyFormat, resourceSkelName)
}

func FormatDescriptorResourcePrefix() string {
	return descriptorResourcePrefix
}

func FormatDescriptorWebKey(webSkelName string) string {
	return fmt.Sprintf("descriptor:web:%s", webSkelName)
}

func FormatDescriptorWebPrefix() string {
	return "descriptor:web"
}

// DescriptorDomain is the domain contract published through Hub Watch.
type DescriptorDomain = skeldesc.Domain

// FormatDescriptorDomainKey returns the Watch key for a domain descriptor.
func FormatDescriptorDomainKey(domainName string) string {
	return fmt.Sprintf("descriptor:domain:%s", domainName)
}

// FormatDescriptorDomainPrefix returns the domain descriptor Watch prefix.
func FormatDescriptorDomainPrefix() string {
	return "descriptor:domain"
}

// DescriptorData is the data contract published through Hub Watch.
type DescriptorData = skeldesc.Data

// FormatDescriptorDataKey returns the Watch key for a data descriptor.
func FormatDescriptorDataKey(dataSkelName string) string {
	return fmt.Sprintf("descriptor:data:%s", dataSkelName)
}

// FormatDescriptorDataPrefix returns the data descriptor Watch prefix.
func FormatDescriptorDataPrefix() string {
	return "descriptor:data"
}

// DescriptorEnum is the enum contract published through Hub Watch.
type DescriptorEnum = skeldesc.Enum

// FormatDescriptorEnumKey returns the Watch key for an enum descriptor.
func FormatDescriptorEnumKey(enumSkelName string) string {
	return fmt.Sprintf("descriptor:enum:%s", enumSkelName)
}

// FormatDescriptorEnumPrefix returns the enum descriptor Watch prefix.
func FormatDescriptorEnumPrefix() string {
	return "descriptor:enum"
}

// DescriptorConfig is the config contract published through Hub Watch.
type DescriptorConfig = skeldesc.Config

// FormatDescriptorConfigKey returns the Watch key for a config descriptor.
func FormatDescriptorConfigKey(configSkelName string) string {
	return fmt.Sprintf("descriptor:config:%s", configSkelName)
}

// FormatDescriptorConfigPrefix returns the config descriptor Watch prefix.
func FormatDescriptorConfigPrefix() string {
	return "descriptor:config"
}

// DescriptorEvent is the event contract published through Hub Watch.
type DescriptorEvent = skeldesc.Event

// FormatDescriptorEventKey returns the Watch key for an event descriptor.
func FormatDescriptorEventKey(eventSkelName string) string {
	return fmt.Sprintf("descriptor:event:%s", eventSkelName)
}

// FormatDescriptorEventPrefix returns the event descriptor Watch prefix.
func FormatDescriptorEventPrefix() string {
	return "descriptor:event"
}

// DescriptorTask is the task contract published through Hub Watch.
type DescriptorTask = skeldesc.Task

// FormatDescriptorTaskKey returns the Watch key for a task descriptor.
func FormatDescriptorTaskKey(taskSkelName string) string {
	return fmt.Sprintf("descriptor:task:%s", taskSkelName)
}

// FormatDescriptorTaskPrefix returns the task descriptor Watch prefix.
func FormatDescriptorTaskPrefix() string {
	return "descriptor:task"
}
