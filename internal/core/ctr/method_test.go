package ctr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type methodTestEngine struct {
	stopped bool
}

func (e *methodTestEngine) Start() {}

func (e *methodTestEngine) Stop() {
	e.stopped = true
}

func TestGetMethodByName(t *testing.T) {
	engineType := reflect.TypeOf(&methodTestEngine{})

	method := getMethodByName(engineType, "Stop")
	assert.Equal(t, "Stop", method.Name)

	cachedMethod := getMethodByName(engineType, "Stop")
	assert.Equal(t, method, cachedMethod)
}

func TestGetMethodByNamePanicsForMissingMethod(t *testing.T) {
	assert.PanicsWithError(t, "method=Missing not found in type=", func() {
		getMethodByName(reflect.TypeOf(&methodTestEngine{}), "Missing")
	})
}
