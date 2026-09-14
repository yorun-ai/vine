package core

import "strings"

// FieldSource describes the layers that supplied a field and any variables
// referenced by its seed template. Variable values are never stored in this metadata.
type FieldSource struct {
	Source    string   `json:"source" yaml:"source"`
	Define    string   `json:"define,omitempty" yaml:"define,omitempty"`
	Override  string   `json:"override,omitempty" yaml:"override,omitempty"`
	Variables []string `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type FieldSources map[string]FieldSource

func cloneFieldSources(sources FieldSources) FieldSources {
	result := FieldSources{}
	for path, source := range sources {
		source.Variables = append([]string(nil), source.Variables...)
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
		sources[path] = source
	}
	return sources
}
