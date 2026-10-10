// Package desc owns Portal descriptor subscriptions, caches, and API type dependencies.
package desc

import (
	"context"
	"sync"

	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hubwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
)

var descLogger = logger.New("daemon:portal:desc")

// Manager maintains the single descriptor cache used by admission and inspection.
type Manager struct {
	app.BaseModule
	Context             context.Context    `inject:""`
	Watch               hubwatch.ClientOps `inject:""`
	mutex               sync.RWMutex
	admissionMutex      sync.Mutex
	watchContext        context.Context
	watchCancel         context.CancelFunc
	actorNamesByKey     map[string]string
	serviceNamesByKey   map[string]string
	webNamesByKey       map[string]string
	resourceNamesByKey  map[string]string
	actorsBySkelName    map[string]*watched.DescriptorActor
	servicesBySkelName  map[string]*watched.DescriptorService
	websBySkelName      map[string]*watched.DescriptorWeb
	resourcesBySkelName map[string]*watched.DescriptorResource
	typesByKey          map[string]*_TypeDescriptor
	typeInputs          map[string]*_TypeDescriptor
	typeServices        map[string]*watched.DescriptorService
	generation          uint64
	syncedGeneration    uint64
	syncWake            chan struct{}
	syncDone            chan struct{}
	stopped             bool
	typeSubscription    *_TypeSubscription
	admissionObservers  map[*_AdmissionObserver]struct{}
}

// DIInit loads descriptors and establishes their subscriptions.
func (m *Manager) DIInit() {
	m.watchContext, m.watchCancel = context.WithCancel(m.Context)
	m.actorNamesByKey = map[string]string{}
	m.serviceNamesByKey = map[string]string{}
	m.webNamesByKey = map[string]string{}
	m.resourceNamesByKey = map[string]string{}
	m.actorsBySkelName = map[string]*watched.DescriptorActor{}
	m.servicesBySkelName = map[string]*watched.DescriptorService{}
	m.websBySkelName = map[string]*watched.DescriptorWeb{}
	m.resourcesBySkelName = map[string]*watched.DescriptorResource{}
	m.typesByKey = map[string]*_TypeDescriptor{}
	m.typeInputs = map[string]*_TypeDescriptor{}
	m.syncWake = make(chan struct{}, 1)
	m.syncDone = make(chan struct{})
	m.admissionObservers = map[*_AdmissionObserver]struct{}{}
	m.loadActors()
	m.loadServices()
	m.loadWebs()
	m.loadResources()
	// Keep the initial healthy snapshot available at the end of initialization.
	// Transient type failures are retried by the worker without disabling admission.
	if err := m.syncTypes(); err != nil {
		descLogger.Warn("descriptor type synchronization failed", "error", err)
	}
	go m.runTypeSync()
}
