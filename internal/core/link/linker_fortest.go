package link

import "go.yorun.ai/vine/internal/core/meta"

// SetNewLinkerForTest replaces the application Linker factory and returns its restore function.
func SetNewLinkerForTest(factory func(app meta.App, endpoint string) Linker) func() {
	prev := newLinker
	newLinker = factory

	return func() {
		newLinker = prev
	}
}
