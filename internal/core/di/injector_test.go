package di

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MailGateway interface {
	Provider() string
}

type SmtpGateway struct {
	SingletonScoped
}

func (g *SmtpGateway) Provider() string {
	return "smtp"
}

type AuditLogger struct {
	SingletonScoped
	initialized bool
}

func (l *AuditLogger) DIInit() {
	l.initialized = true
}

type RequestIdentity interface {
	UserID() string
}

type SessionIdentity struct {
	ExecutionScoped
	userID string
}

func (i *SessionIdentity) UserID() string {
	return i.userID
}

type NotificationService interface {
	Channel() string
}

type EmailNotificationService struct {
	SingletonScoped
	Logger  *AuditLogger `inject:""`
	Gateway MailGateway  `inject:""`
}

func (s *EmailNotificationService) Channel() string {
	return s.Gateway.Provider()
}

type RequestHandler struct {
	ExecutionScoped
	Notifier NotificationService `inject:""`
	Identity RequestIdentity     `inject:""`
}

var cleanupOrder []string

type AppCache struct {
	SingletonScoped
}

func (c *AppCache) DIDispose() {
	cleanupOrder = append(cleanupOrder, "singleton")
}

type RequestScope struct {
	ExecutionScoped
}

func (s *RequestScope) DIDispose() {
	cleanupOrder = append(cleanupOrder, "execution")
}

type RequestMetrics struct {
	ExecutionScoped
}

type RequestTrace struct {
	TransientScoped
}

func (t *RequestTrace) DIDispose() {
	cleanupOrder = append(cleanupOrder, "transient")
}

type RequestPipeline struct {
	ExecutionScoped
	Cache   *AppCache       `inject:""`
	Scope   *RequestScope   `inject:""`
	Metrics *RequestMetrics `inject:""`
	Trace   *RequestTrace   `inject:""`
}

type GlobalConfig struct {
	SingletonScoped
}

type SeededRequestContext struct {
	ExecutionScoped
}

func (c *SeededRequestContext) DIDispose() {
	cleanupOrder = append(cleanupOrder, "seeded-request")
}

type assignableSeedContext struct {
	ExecutionScoped
}

type implicitOnlyLeaf struct {
	SingletonScoped
}

type implicitOnlyRoot struct {
	SingletonScoped
	Leaf *implicitOnlyLeaf `inject:""`
}

type implicitInterfaceDep interface {
	Run() string
}

type implicitInterfaceRoot struct {
	SingletonScoped
	Dependency implicitInterfaceDep `inject:""`
}

type parentProvidedDependency struct {
	SingletonScoped
}

type childImplicitLeaf struct {
	SingletonScoped
}

type childImplicitRoot struct {
	SingletonScoped
	Parent *parentProvidedDependency `inject:""`
	Leaf   *childImplicitLeaf        `inject:""`
}

type injectorAwareSingleton struct {
	SingletonScoped
	Injector Injector `inject:""`
}

type injectorAwareExecution struct {
	ExecutionScoped
	Injector Injector `inject:""`
}

type abstractConfig interface {
	Key() string
}

type abstractConfigA struct {
	key string
}

func (c *abstractConfigA) Key() string {
	return c.key
}

type abstractConfigB struct {
	key string
}

func (c *abstractConfigB) Key() string {
	return c.key
}

type abstractOther interface {
	Key() string
}

type _AbstractDisposable interface {
	Key() string
}

type _AbstractDisposableImpl struct {
	disposed *bool
}

func (*_AbstractDisposableImpl) Key() string {
	return "disposable"
}

func (i *_AbstractDisposableImpl) DIDispose() {
	*i.disposed = true
}

type EmbeddedPointerInjectedBase struct {
	Dependency *RequestScope `inject:""`
}

type EmbeddedPointerInjectedRoot struct {
	SingletonScoped
	*EmbeddedPointerInjectedBase
}

var abstractFactoryCalls int

func TestInjectorResolvesDependenciesAcrossScopes(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[MailGateway]()).ToImplementation(T[*SmtpGateway]()).In(SingletonScope)
		b.Bind(T[NotificationService]()).ToImplementation(T[*EmailNotificationService]()).In(SingletonScope)
		b.Bind(T[RequestIdentity]()).ToImplementation(T[*SessionIdentity]()).In(ExecutionScope)
		b.Bind(T[*RequestHandler]())
	})

	execution := injector.StartExecution(func(s *Seeder) {
		s.Seed(T[RequestIdentity](), &SessionIdentity{userID: "alice"})
	})

	var handler *RequestHandler
	execution.Resolve(&handler)

	assert.NotNil(t, handler)
	assert.Equal(t, "alice", handler.Identity.UserID())
	assert.Equal(t, "smtp", handler.Notifier.Channel())

	notifierImpl := handler.Notifier.(*EmailNotificationService)
	assert.True(t, notifierImpl.Logger.initialized)
}

func TestInjectorResolveInjectsPlainInjectorIntoInjectorInterface(t *testing.T) {
	injector := NewInjector()

	var resolved Injector
	injector.Resolve(&resolved)

	plain, ok := resolved.(*_PlainInjector)
	if assert.True(t, ok) {
		assert.Same(t, injector, plain)
	}
}

func TestExecutionResolveInjectsExecutionInjectorIntoInjectorInterface(t *testing.T) {
	injector := NewInjector()
	execution := injector.StartExecution()

	var resolved Injector
	execution.Resolve(&resolved)

	executing, ok := resolved.(*_ExecutionInjector)
	if assert.True(t, ok) {
		assert.Same(t, execution, executing)
	}
}

func TestInjectorInterfaceDependencyUsesCurrentInjectorContext(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*injectorAwareSingleton]())
		b.Bind(T[*injectorAwareExecution]())
	})

	var singleton *injectorAwareSingleton
	injector.Resolve(&singleton)
	if assert.NotNil(t, singleton) {
		plain, ok := singleton.Injector.(*_PlainInjector)
		if assert.True(t, ok) {
			assert.Same(t, injector, plain)
		}
	}

	execution := injector.StartExecution()

	var executionAware *injectorAwareExecution
	execution.Resolve(&executionAware)
	if assert.NotNil(t, executionAware) {
		executing, ok := executionAware.Injector.(*_ExecutionInjector)
		if assert.True(t, ok) {
			assert.Same(t, execution, executing)
		}
	}
}

func TestInjectorFallbackFactoryResolvesUnboundConcreteTypes(t *testing.T) {
	abstractFactoryCalls = 0
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			abstractFactoryCalls++
			switch ctx.TargetType {
			case T[*abstractConfigA]():
				return &abstractConfigA{key: "a"}
			case T[*abstractConfigB]():
				return &abstractConfigB{key: "b"}
			default:
				panic("unexpected target type")
			}
		}).In(SingletonScope)
	})

	var first *abstractConfigA
	var second *abstractConfigA
	var other *abstractConfigB
	injector.Resolve(&first)
	injector.Resolve(&second)
	injector.Resolve(&other)

	if assert.NotNil(t, first) && assert.NotNil(t, second) {
		assert.Same(t, first, second)
		assert.Equal(t, "a", first.Key())
	}
	if assert.NotNil(t, other) {
		assert.Equal(t, "b", other.Key())
	}
	assert.Equal(t, 2, abstractFactoryCalls)
}

func TestAbstractSingletonIsSharedAcrossPlainSubAndExecutionInjectors(t *testing.T) {
	abstractFactoryCalls = 0
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			abstractFactoryCalls++
			return &abstractConfigA{key: ctx.TargetType.String()}
		}).In(SingletonScope)
	})
	subInjector := injector.SubInjector()
	firstExecution := injector.StartExecution()
	secondExecution := subInjector.StartExecution()

	var root *abstractConfigA
	var sub *abstractConfigA
	var first *abstractConfigA
	var second *abstractConfigA
	injector.Resolve(&root)
	subInjector.Resolve(&sub)
	firstExecution.Resolve(&first)
	secondExecution.Resolve(&second)

	assert.Same(t, root, sub)
	assert.Same(t, root, first)
	assert.Same(t, root, second)
	assert.Equal(t, 1, abstractFactoryCalls)
}

func TestAbstractSingletonResolvedByExecutionRemainsOwnedByPlainInjector(t *testing.T) {
	disposed := false
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[_AbstractDisposable]()).ToAbstractFactory(func(ResolveContext) _AbstractDisposable {
			return &_AbstractDisposableImpl{disposed: &disposed}
		}).In(SingletonScope)
	})
	execution := injector.StartExecution()

	var fromExecution *_AbstractDisposableImpl
	execution.Resolve(&fromExecution)
	execution.CompleteExecution()

	var fromPlain *_AbstractDisposableImpl
	injector.Resolve(&fromPlain)

	assert.Same(t, fromExecution, fromPlain)
	assert.False(t, disposed)
}

func TestInjectorFallbackFactorySupportsConcurrentFirstResolution(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			switch ctx.TargetType {
			case T[*abstractConfigA]():
				return &abstractConfigA{key: "a"}
			case T[*abstractConfigB]():
				return &abstractConfigB{key: "b"}
			default:
				panic("unexpected target type")
			}
		}).In(SingletonScope)
	})

	var wg sync.WaitGroup
	start := make(chan struct{})
	missing := make(chan reflect.Type, 32)
	for index := 0; index < 32; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			if index%2 == 0 {
				var resolved *abstractConfigA
				injector.Resolve(&resolved)
				if resolved == nil {
					missing <- T[*abstractConfigA]()
				}
				return
			}
			var resolved *abstractConfigB
			injector.Resolve(&resolved)
			if resolved == nil {
				missing <- T[*abstractConfigB]()
			}
		}(index)
	}
	close(start)
	wg.Wait()
	close(missing)
	assert.Empty(t, missing)
}

func TestInjectorExplicitBindingOverridesFallbackFactory(t *testing.T) {
	abstractFactoryCalls = 0
	explicit := &abstractConfigA{key: "explicit"}
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			abstractFactoryCalls++
			return &abstractConfigA{key: ctx.TargetType.String()}
		}).In(SingletonScope)
		b.Bind(T[*abstractConfigA]()).ToInstance(explicit)
	})

	var resolved *abstractConfigA
	injector.Resolve(&resolved)

	assert.Same(t, explicit, resolved)
	assert.Equal(t, 0, abstractFactoryCalls)
}

func TestAbstractExecutionScopeReusesPerConcreteTypeWithinExecution(t *testing.T) {
	abstractFactoryCalls = 0
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			abstractFactoryCalls++
			return &abstractConfigA{key: ctx.TargetType.String()}
		}).In(ExecutionScope)
	})

	firstExecution := injector.StartExecution()
	secondExecution := injector.StartExecution()

	var first *abstractConfigA
	var second *abstractConfigA
	var third *abstractConfigA
	firstExecution.Resolve(&first)
	firstExecution.Resolve(&second)
	secondExecution.Resolve(&third)

	if assert.NotNil(t, first) && assert.NotNil(t, second) {
		assert.Same(t, first, second)
	}
	if assert.NotNil(t, first) && assert.NotNil(t, third) {
		assert.NotSame(t, first, third)
	}
	assert.Equal(t, 2, abstractFactoryCalls)
}

func TestAbstractExecutionScopeCannotResolveFromPlainInjector(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			return &abstractConfigA{key: ctx.TargetType.String()}
		}).In(ExecutionScope)
	})

	assert.Panics(t, func() {
		var config *abstractConfigA
		injector.Resolve(&config)
	})
}

func TestAbstractTransientScopeCreatesNewInstancePerRequest(t *testing.T) {
	abstractFactoryCalls = 0
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ctx ResolveContext) abstractConfig {
			abstractFactoryCalls++
			return &abstractConfigA{key: ctx.TargetType.String()}
		}).In(TransientScope)
	})

	var first *abstractConfigA
	var second *abstractConfigA
	injector.Resolve(&first)
	injector.Resolve(&second)

	if assert.NotNil(t, first) && assert.NotNil(t, second) {
		assert.NotSame(t, first, second)
	}
	assert.Equal(t, 2, abstractFactoryCalls)
}

func TestInjectorAbstractRejectsDirectInterfaceRequest(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ResolveContext) abstractConfig {
			return &abstractConfigA{key: "a"}
		}).In(SingletonScope)
	})

	assert.PanicsWithError(t,
		"abstract binding di.abstractConfig cannot resolve direct interface requests",
		func() {
			var config abstractConfig
			injector.Resolve(&config)
		},
	)
}

func TestInjectorAbstractRejectsMultipleMatches(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[abstractConfig]()).ToAbstractFactory(func(ResolveContext) abstractConfig {
			return &abstractConfigA{key: "a"}
		}).In(SingletonScope)
		b.Bind(T[abstractOther]()).ToAbstractFactory(func(ResolveContext) abstractOther {
			return &abstractConfigA{key: "other"}
		}).In(SingletonScope)
	})

	assert.PanicsWithError(t,
		"multiple abstract bindings matched for *di.abstractConfigA",
		func() {
			var config *abstractConfigA
			injector.Resolve(&config)
		},
	)
}

func TestInjectorCompleteImplicitsAddsMissingLocalBindingsAndBounds(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*implicitOnlyRoot]())
	}).(*_PlainInjector)

	rootBinding := injector.bindings[T[*implicitOnlyRoot]()]
	leafBinding := injector.bindings[T[*implicitOnlyLeaf]()]
	rootBound := injector.bounds[T[*implicitOnlyRoot]()]
	leafBound := injector.bounds[T[*implicitOnlyLeaf]()]

	if assert.NotNil(t, rootBinding) {
		assert.False(t, rootBinding.isImplicit)
	}
	if assert.NotNil(t, leafBinding) {
		assert.True(t, leafBinding.isImplicit)
	}
	assert.NotNil(t, rootBound)
	assert.NotNil(t, leafBound)
}

func TestInjectorFallbackScopeAppliesToUnsetBindingsAndImplicits(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*scopeNoMarker]())
		b.Bind(T[*implicitOnlyRoot]())
	}).(*_PlainInjector)

	assert.Equal(t, noScope, injector.bounds[T[*scopeNoMarker]()].DeclaredScope())
	assert.Equal(t, TransientScope, injector.bounds[T[*scopeNoMarker]()].ResolveScope(injector.fallbackScope))
	assert.Equal(t, SingletonScope, injector.bounds[T[*implicitOnlyRoot]()].ResolveScope(injector.fallbackScope))
	assert.Equal(t, SingletonScope, injector.bounds[T[*implicitOnlyLeaf]()].ResolveScope(injector.fallbackScope))
}

func TestSubInjectorCompleteImplicitsDoesNotDuplicateParentBindings(t *testing.T) {
	parent := NewInjector(func(b *Binder) {
		b.Bind(T[*parentProvidedDependency]())
	}).(*_PlainInjector)

	child := parent.SubInjector(func(b *Binder) {
		b.Bind(T[*childImplicitRoot]())
	}).(*_PlainInjector)

	assert.NotNil(t, child.bindings[T[*childImplicitRoot]()])
	assert.NotNil(t, child.bindings[T[*childImplicitLeaf]()])
	assert.Nil(t, child.bindings[T[*parentProvidedDependency]()])
	assert.NotNil(t, child.lookupBinding(T[*parentProvidedDependency]()))
}

func TestSubInjectorUsesTransientFallbackScope(t *testing.T) {
	parent := NewInjector().(*_PlainInjector)
	child := parent.SubInjector().(*_PlainInjector)

	assert.Equal(t, TransientScope, child.fallbackScope)
}

func TestExecutionFallbackScopeAppliesToUnsetBindings(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*scopeNoMarker]())
	})

	execution := injector.StartExecution().(*_ExecutionInjector)
	resolver, ok := execution.GetResolver(T[*scopeNoMarker]()).(*_PersistedResolver)
	if assert.True(t, ok) {
		assert.Equal(t, ExecutionScope, resolver.bound.ResolveScope(execution.fallbackScope))
	}
}

func TestInjectorInterfaceIsNotRegisteredAsImplicitBinding(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*injectorAwareSingleton]())
	}).(*_PlainInjector)

	assert.Nil(t, injector.bindings[injectorType])
	assert.NotNil(t, injector.resolvers[injectorType])
}

func TestInjectorRejectsImplicitNonStructDependency(t *testing.T) {
	assert.PanicsWithError(t,
		"implicit binding only supports struct pointer, but di.implicitInterfaceDep found",
		func() {
			_ = NewInjector(func(b *Binder) {
				b.Bind(T[*implicitInterfaceRoot]())
			})
		},
	)
}

func TestInjectorRejectsEmbeddedPointerStructWithInjectedFields(t *testing.T) {
	assert.PanicsWithError(t,
		"embedded pointer struct EmbeddedPointerInjectedBase of EmbeddedPointerInjectedRoot contains injected fields, use value embedding instead",
		func() {
			_ = NewInjector(func(b *Binder) {
				b.Bind(T[*EmbeddedPointerInjectedRoot]())
			})
		},
	)
}

func TestExecutionDisposalRunsInReverseCreationOrder(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*AppCache]())
		b.Bind(T[*RequestScope]())
		b.Bind(T[*RequestMetrics]()).WithDisposer(func(*RequestMetrics) {
			cleanupOrder = append(cleanupOrder, "func")
		})
		b.Bind(T[*RequestTrace]())
		b.Bind(T[*RequestPipeline]())
	})

	execution := injector.StartExecution()

	var pipeline *RequestPipeline
	execution.Resolve(&pipeline)
	assert.NotNil(t, pipeline)

	execution.CompleteExecution()

	assert.Equal(t, []string{"transient", "func", "execution"}, cleanupOrder)
}

func TestPersistedDisposeRunsOnceForExecutionScopedInstance(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*RequestScope]())
	})

	execution := injector.StartExecution()

	var first *RequestScope
	var second *RequestScope
	execution.Resolve(&first)
	execution.Resolve(&second)

	assert.Same(t, first, second)

	execution.CompleteExecution()

	assert.Equal(t, []string{"execution"}, cleanupOrder)
}

func TestNonExecutionInjectorCannotResolveExecutionScopedType(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*RequestScope]())
	})

	assert.Panics(t, func() {
		var scope *RequestScope
		injector.Resolve(&scope)
	})

	subInjector := injector.SubInjector()
	assert.Panics(t, func() {
		var scope *RequestScope
		subInjector.Resolve(&scope)
	})
}

func TestSeederRequiresExecutionScope(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*GlobalConfig]())
		b.Bind(T[*RequestScope]())
	})

	assert.NotPanics(t, func() {
		execution := injector.StartExecution(func(seeder *Seeder) {
			seeder.SeedInstance(&RequestScope{})
		})
		assert.IsType(t, &_ExecutionInjector{}, execution)
	})

	assert.Panics(t, func() {
		injector.StartExecution(func(seeder *Seeder) {
			seeder.SeedInstance(&GlobalConfig{})
		})
	})
}

func TestSeedDoesNotInstantiateImmediately(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*SeededRequestContext]())
	})

	execution := injector.StartExecution(func(seeder *Seeder) {
		seeder.SeedInstance(&SeededRequestContext{})
	})

	assert.Empty(t, cleanupOrder)

	execution.CompleteExecution()

	assert.Empty(t, cleanupOrder)
}

func TestSeededInstanceDoesNotParticipateInDispose(t *testing.T) {
	cleanupOrder = nil
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*SeededRequestContext]())
	})

	execution := injector.StartExecution(func(seeder *Seeder) {
		seeder.SeedInstance(&SeededRequestContext{})
	})

	var context *SeededRequestContext
	execution.Resolve(&context)

	assert.NotNil(t, context)

	execution.CompleteExecution()

	assert.Empty(t, cleanupOrder)
}

func TestSeedPanicsAfterExecutionScopedInstanceResolved(t *testing.T) {
	injector := NewInjector(func(b *Binder) {
		b.Bind(T[*RequestScope]())
	})

	var seeder *Seeder
	execution := injector.StartExecution(func(current *Seeder) {
		seeder = current
	})

	var scope *RequestScope
	execution.Resolve(&scope)
	assert.NotNil(t, scope)

	assert.PanicsWithError(t, "execution injector is no longer accepting seeds", func() {
		seeder.SeedInstance(&RequestScope{})
	})
}

func TestSeedRejectsIncompatibleConcreteInstanceWithFriendlyError(t *testing.T) {
	targetType := reflect.TypeOf(&struct {
		ExecutionScoped
	}{})

	injector := NewInjector(func(b *Binder) {
		b.Bind(targetType)
	})

	assert.PanicsWithError(t,
		"instance type *di.assignableSeedContext is not compatible with *struct { di.ExecutionScoped }",
		func() {
			injector.StartExecution(func(seeder *Seeder) {
				seeder.Seed(targetType, &assignableSeedContext{})
			})
		},
	)
}
