// Package ex defines Vine error codes, structured errors, panic helpers, and
// recovery functions.
//
// # Managed panic boundaries
//
// Vine automatically recovers panics raised while it invokes these application
// entry points:
//
//   - Rpc service methods
//   - Event listener methods
//   - Task runner methods
//   - Web handler methods
//
// At Rpc, Event, and Task boundaries, a recovered Error continues through the
// normal structured-error path. A non-Error panic is logged with its stack and
// becomes Internal.
//
// At a Web boundary, an Error is mapped to its canonical HTTP status when the
// response has not started. Vine does not impose an error response body, so Web
// handlers remain free to define HTML, JSON, streaming, or other HTTP response
// contracts. If the response has started, Vine preserves it and only aborts the
// remaining handler chain.
//
// A non-OK Error retains a local diagnostic stack for server-side logging.
// Wrapping preserves a cause's stack, and panic recovery fills it only when the
// Error has no stack yet. The stack and local panic diagnostics are never
// included in a serialized error or Web response. A non-Error Web panic is
// logged with its stack and returns HTTP status 500 if the response has not
// started.
//
// The following execution paths do not use that structured-error boundary:
//
//   - Application and module lifecycle hooks are not recovered as business
//     errors.
//   - Goroutines started by application code are not automatically recovered.
//
// Code outside a managed boundary must return errors normally or establish an
// explicit boundary with Recover or RecoverApplication. In particular, a panic
// in an application-owned goroutine can terminate the process if the goroutine
// does not recover it.
package ex
