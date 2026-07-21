package minder

func (m *AppMinder) AddMutator(mutator Mutator) {
	m.mutex.Lock()
	m.mutators = append(m.mutators, mutator)
	m.mutex.Unlock()
}

func (m *AppMinder) appInstance(instanceID string) (*AppInstance, bool) {
	m.mutex.Lock()
	instance := m.instanceByID[instanceID]
	m.mutex.Unlock()
	if instance == nil {
		return nil, false
	}
	return instance, true
}

func (m *AppMinder) appInstances() []*AppInstance {
	m.mutex.Lock()
	instances := make([]*AppInstance, 0, len(m.instanceByID))
	for _, instance := range m.instanceByID {
		instances = append(instances, instance)
	}
	m.mutex.Unlock()
	return instances
}

func (m *AppMinder) addInstance(instance *AppInstance) bool {
	m.mutex.Lock()
	if m.instanceByID[instance.AppInfo.InstanceId()] != nil {
		m.mutex.Unlock()
		return false
	}

	m.instanceByID[instance.AppInfo.InstanceId()] = instance
	m.mutex.Unlock()

	return true
}

func (m *AppMinder) removeInstance(instance *AppInstance) {
	m.mutex.Lock()
	if _, ok := m.instanceByID[instance.AppInfo.InstanceId()]; !ok {
		m.mutex.Unlock()
		return
	}

	delete(m.instanceByID, instance.AppInfo.InstanceId())
	m.mutex.Unlock()
}

func (m *AppMinder) beforeRegistration(instance *AppInstance) {
	m.mutex.Lock()
	mutators := append([]Mutator(nil), m.mutators...)
	m.mutex.Unlock()

	for _, mutator := range mutators {
		mutator.OnSetup(instance)
	}
}

func (m *AppMinder) onDrain(instance *AppInstance) {
	m.mutex.Lock()
	mutators := append([]Mutator(nil), m.mutators...)
	m.mutex.Unlock()

	for _, mutator := range mutators {
		mutator.OnDrain(instance)
	}
}

func (m *AppMinder) onDestroy(instance *AppInstance) {
	m.mutex.Lock()
	mutators := append([]Mutator(nil), m.mutators...)
	m.mutex.Unlock()

	for _, mutator := range mutators {
		mutator.OnDestroy(instance)
	}
}
