package app

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testProvidedFlag struct {
	FlagModel
	Value string
}

func TestFlagsEnsureRunFlagAddsDefaultRunFlag(t *testing.T) {
	flags := _Flags{}

	flags.EnsureRunFlag()

	runFlag, ok := flags[T[*RunFlag]()].(*RunFlag)
	if assert.True(t, ok) {
		assert.Equal(t, "", runFlag.ListenAddr)
		assert.Nil(t, runFlag.Context)
	}
}

func TestFlagsEnsureRunFlagKeepsProvidedRunFlag(t *testing.T) {
	runFlag := &RunFlag{
		ListenAddr: ":18080",
	}
	flags := _Flags{
		T[*RunFlag](): runFlag,
	}

	flags.EnsureRunFlag()

	assert.Equal(t, ":18080", flags[T[*RunFlag]()].(*RunFlag).ListenAddr)
}

func TestFlagsListenAddrReturnsRunFlagValue(t *testing.T) {
	flags := _Flags{
		T[*RunFlag](): &RunFlag{ListenAddr: ":28089"},
	}

	assert.Equal(t, ":28089", flags.ListenAddr())
}

func TestFlagsContextReturnsRunFlagContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "k", "v")
	flags := _Flags{
		T[*RunFlag](): &RunFlag{Context: ctx},
	}

	assert.Same(t, ctx, flags.Context())
}

func TestFlagsContextReturnsBackgroundWhenRunFlagContextIsNil(t *testing.T) {
	flags := _Flags{
		T[*RunFlag](): &RunFlag{},
	}

	assert.NotNil(t, flags.Context())
}

func TestFlagsLinkEndpointReturnsRunFlagValue(t *testing.T) {
	flags := _Flags{}
	flags.EnsureRunFlag()

	assert.Empty(t, flags.LinkEndpoint())
}

func TestWithLinkEndpointSetsRunFlag(t *testing.T) {
	flags := _Flags{}

	flags.Apply(WithLinkEndpoint("http://127.0.0.1:7079"))

	assert.Equal(t, "http://127.0.0.1:7079", flags.LinkEndpoint())
}

func TestFlagsApplySetsListenAddr(t *testing.T) {
	flags := _Flags{}
	flag := &RunFlag{ListenAddr: ":28089"}

	flags.Apply(With(flag))

	if assert.Len(t, flags, 1) {
		assert.Same(t, flag, flags[reflect.TypeOf(flag)])
	}
}

func TestFlagsApplyAddsProvidedBindingByConcreteType(t *testing.T) {
	flags := _Flags{}
	flag := &testProvidedFlag{Value: "demo"}

	flags.Apply(With(flag))

	if assert.Len(t, flags, 1) {
		assert.Same(t, flag, flags[reflect.TypeOf(flag)])
	}
}

func TestWithPanicsForNonStructPointer(t *testing.T) {
	assert.PanicsWithError(t, "flag app.testProvidedFlag must be pointer to struct", func() {
		_ = With(testProvidedFlag{})
	})
}

func TestWithPanicsForNil(t *testing.T) {
	var flag Flag
	assert.PanicsWithError(t, "flag must not be nil", func() {
		_ = With(flag)
	})
}

func TestFlagsApplyPanicsOnDuplicateProvidedType(t *testing.T) {
	flags := _Flags{}

	assert.PanicsWithError(t, "type *app.testProvidedFlag was already provided", func() {
		flags.Apply(
			With(&testProvidedFlag{Value: "a"}),
			With(&testProvidedFlag{Value: "b"}),
		)
	})
}
