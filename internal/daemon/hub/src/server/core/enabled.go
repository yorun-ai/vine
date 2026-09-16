package core

// EnabledOrDefault returns the enable switch a creation request asks for. Hub
// publishes only enabled configuration, so a request that omits the switch
// enables the entity.
func EnabledOrDefault(value *bool) bool {
	return value == nil || *value
}
