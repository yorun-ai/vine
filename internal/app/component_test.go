package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testComponent struct {
	BaseComponent
	userComponent ManagedComponent
}

type testManagedComponent struct {
	BaseManagedComponent[*testManagedComponent]
	BaseComponentManager
	userComponent ManagedComponent
}

func (c *testManagedComponent) InitComponent(userComponent ManagedComponent) {
	c.userComponent = userComponent
}

func (c *testManagedComponent) Component() ManagedComponent {
	return c.userComponent
}

type testUserComponent struct {
	testManagedComponent
}

type testComponentAppSpec struct {
	Application
}

func (*testComponentAppSpec) Name() string {
	return "test.component"
}

func (*testComponentAppSpec) InitComponents(addComponent TypeAdder) {
	addComponent(T[*testUserComponent]())
}

func TestInitComponentsPassesUserComponentToManagedComponent(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testComponentAppSpec{AppFlag: &RunFlag{}}, flags)
	app.initInjector()

	app.initComponents()

	require.Len(t, app.componentManagers, 1)
	fxComponent, ok := app.componentManagers[0].(*testManagedComponent)
	require.True(t, ok)
	assert.NotNil(t, fxComponent.userComponent)
	_, ok = fxComponent.userComponent.(*testUserComponent)
	assert.True(t, ok)
}
