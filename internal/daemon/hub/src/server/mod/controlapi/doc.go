// Package controlapi serves the Hub Control API that Link and Portal call, on a
// listener of its own.
//
// The package is an entry module: Hub's application spec wires it at the edge,
// and it composes the Control API service implementations. Hub's Admin API and
// Dashboard are deliberately absent from this server and belong to the admin
// module. An entry module may depend on core, repo, and impl, but nothing in
// those packages depends on it: the application owns how Hub is exposed, and the
// layers below stay independent of the listener that exposes them.
package controlapi
