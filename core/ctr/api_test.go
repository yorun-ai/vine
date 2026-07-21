package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.yorun.ai/vine/core/di"
)

type ctrFacadeTarget struct {
	di.SingletonScoped
}

func (t *ctrFacadeTarget) Sum(left int, right int) int {
	return left + right
}

func TestNewContainerFacadeRunsExecution(t *testing.T) {
	container := NewContainer(Option{
		BindAppliers: []di.BindApplier{
			func(b *di.Binder) {
				b.Bind(di.T[*ctrFacadeTarget]())
			},
		},
	})

	method, ok := reflect.TypeOf(&ctrFacadeTarget{}).MethodByName("Sum")
	if !ok {
		t.Fatal("expected Sum method")
	}

	execution := container.NewExecution(reflect.TypeOf(&ctrFacadeTarget{}), method)
	execution.Execute([]any{2, 5})

	assert.Equal(t, []any{7}, execution.Results())
}
