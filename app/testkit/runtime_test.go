package testkit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/core/meta"
)

func TestStartStandaloneStartsRuntime(t *testing.T) {
	registerTestFixtures()

	runtime := StartStandalone[*_TestAppSpec](t, Option{
		ConfigOverrides: []ConfigOverride{
			OverrideConfig(&_TestConfig{DSN: "sqlite://test"}),
		},
	})

	actor := meta.NewAnonymousActor()
	trace := meta.InitialTrace()
	ctx := context.WithValue(context.Background(), "testkit", "execution")
	execution := runtime.NewExecution(ExecutionOption{
		Context: ctx,
		Trace:   trace,
		Actor:   actor,
	})

	require.NotNil(t, execution.newClient())
	assert.Equal(t, "execution", execution.context.Value("testkit"))
	assert.Equal(t, trace, execution.trace)
	assert.Equal(t, actor, execution.actor)

	serviceClient := execution.NewClient[_TestServiceClient]()
	require.NotNil(t, serviceClient)

	erServiceClient := execution.NewClientER[_TestServiceClientER]()
	require.NotNil(t, erServiceClient)
}
