// Package admin serves the Hub Admin API and the embedded Dashboard build on a
// listener of its own.
//
// The package is an entry module: Hub's application spec wires it at the edge,
// and it composes the Admin API service implementations with the Dashboard. An
// entry module may depend on core, repo, and impl, but nothing in those packages
// depends on it: the application owns how Hub is exposed, and the layers below
// stay independent of the listener that exposes them. Move a rule Hub enforces
// into core rather than importing this package from it.
package admin
