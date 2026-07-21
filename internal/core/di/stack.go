package di

import (
	"reflect"
	"strings"
)

type _BuildStack []reflect.Type

func (s _BuildStack) push(t reflect.Type) _BuildStack {
	return append(s, t)
}

func (s _BuildStack) contains(t reflect.Type) bool {
	for _, item := range s {
		if item == t {
			return true
		}
	}
	return false
}

func (s _BuildStack) String() string {
	if len(s) == 0 {
		return "<empty>"
	}

	parts := make([]string, 0, len(s))
	for _, t := range s {
		parts = append(parts, t.String())
	}
	return strings.Join(parts, " -> ")
}
