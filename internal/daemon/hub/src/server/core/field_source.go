package core

import (
	"go.yorun.ai/vine/internal/core/skel"
	"strings"
)

// FieldSource describes the layers that supplied a field and any variables
// referenced by its seed template, including the values actually applied.
type FieldSource struct {
	Source    string               `json:"source" yaml:"source"`
	Define    string               `json:"define,omitempty" yaml:"define,omitempty"`
	Override  string               `json:"override,omitempty" yaml:"override,omitempty"`
	Template  *skel.JSON           `json:"template,omitempty" yaml:"template,omitempty"`
	Bindings  []FieldSourceBinding `json:"bindings,omitempty" yaml:"bindings,omitempty"`
	Variables []string             `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// FieldSourceBinding records a substitution at a JSON pointer within Template.
// Value is the validated value applied at that location, including defaults.
type FieldSourceBinding struct {
	Path        string    `json:"path" yaml:"path"`
	Variable    string    `json:"variable" yaml:"variable"`
	Reference   string    `json:"reference" yaml:"reference"`
	Value       skel.JSON `json:"value" yaml:"value"`
	DefaultUsed bool      `json:"defaultUsed" yaml:"defaultUsed"`
}

type FieldSources map[string]FieldSource

func cloneFieldSources(sources FieldSources) FieldSources {
	result := FieldSources{}
	for path, source := range sources {
		source.Variables = append([]string(nil), source.Variables...)
		if source.Template != nil {
			value := *source.Template
			source.Template = &value
		}
		source.Bindings = append([]FieldSourceBinding(nil), source.Bindings...)
		result[path] = source
	}
	return result
}

// Explicit admin updates replace the affected field's origin even if its value
// is unchanged. Old variable dependencies no longer describe the new input.
func overrideFieldSource(sources FieldSources, path string) FieldSources {
	if sources == nil {
		sources = FieldSources{}
	}
	source, found := sources[path]
	for key := range sources {
		if key == path || strings.HasPrefix(key, path+"/") {
			delete(sources, key)
			found = true
		}
	}
	if found {
		source.Source = "hub"
		source.Override = "hub"
		source.Variables = nil
		source.Template = nil
		source.Bindings = nil
		sources[path] = source
	}
	return sources
}
