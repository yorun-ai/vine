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
//
// At these boundaries, a recovered Error continues through the normal
// structured-error path. A non-Error panic is logged with its stack and becomes
// Internal. A system Error raised through a Panic helper retains its raise stack
// for server-side logging; the stack is never included in the serialized error.
//
// The following execution paths do not use that structured-error boundary:
//
//   - Web handlers use generic HTTP panic recovery and return status 500; an
//     Error panic is not mapped to a structured Rpc error.
//   - Application and module lifecycle hooks are not recovered as business
//     errors.
//   - Goroutines started by application code are not automatically recovered.
//
// Code outside a managed boundary must return errors normally or establish an
// explicit boundary with Recover or RecoverApplication. In particular, a panic
// in an application-owned goroutine can terminate the process if the goroutine
// does not recover it.
package ex
