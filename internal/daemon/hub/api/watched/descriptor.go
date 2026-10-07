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
