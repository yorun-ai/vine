package minder

import "uuid"

func (m *AppMinder) AddMutator(mutator Mutator) {
	m.mutex.Lock()
	m.mutators = append(m.mutators, mutator)
	m.mutex.Unlock()
}

func (m *AppMinder) appInstance(instanceID uuid.UUID) (*AppInstance, bool) {
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

func (m *AppMinder) addInstance(instance *AppInstance) *AppInstance {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	id := instance.AppInfo.InstanceId()
	if existing := m.instanceByID[id]; existing != nil {
		return existing
	}
	m.instanceByID[id] = instance
	return instance
}

func (m *AppMinder) removeInstance(instance *AppInstance) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	id := instance.AppInfo.InstanceId()
	if m.instanceByID[id] == instance {
		delete(m.instanceByID, id)
	}
}

func (m *AppMinder) mutatorSnapshot() []Mutator {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]Mutator(nil), m.mutators...)
}
