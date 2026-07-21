package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testComponent struct {
	BaseComponent
	userComponent FrameworkComponent
}

type testFrameworkComponent struct {
	BaseFrameworkComponent[*testFrameworkComponent]
	BaseFrameworkComponentMinder
	userComponent FrameworkComponent
}

func (c *testFrameworkComponent) InitComponent(userComponent FrameworkComponent) {
	c.userComponent = userComponent
}

func (c *testFrameworkComponent) Component() FrameworkComponent {
	return c.userComponent
}

type testUserComponent struct {
	testFrameworkComponent
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

func TestInitComponentsPassesUserComponentToFrameworkComponent(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()
	flags.InitInprocFlag(false)
	app := newApp(&testComponentAppSpec{Application: Application{AppFlag: &RunFlag{}}}, flags)
	app.initInjector()

	app.initComponents()

	require.Len(t, app.frameworkComponentMinders, 1)
	fxComponent, ok := app.frameworkComponentMinders[0].(*testFrameworkComponent)
	require.True(t, ok)
	assert.NotNil(t, fxComponent.userComponent)
	_, ok = fxComponent.userComponent.(*testUserComponent)
	assert.True(t, ok)
}
