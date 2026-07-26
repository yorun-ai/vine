package redact

import internalredact "go.yorun.ai/vine/internal/core/redact"

// Limits bound the work and output produced while rendering a value.
type Limits = internalredact.Limits

// Option controls how a value is projected and rendered.
type Option = internalredact.Option

// Result is the JSON-safe rendering of a value.
type Result = internalredact.Result

// DefaultLimits returns the limits used when Option.Limits is nil.
func DefaultLimits() Limits {
	return internalredact.DefaultLimits()
}

// Render converts value to JSON while masking explicitly sensitive values and
// fields tagged skel:"sensitive". Binary values are always replaced with a
// length summary, including when RevealSensitive is enabled. Traversal and
// output limits remain active in every mode. Failures are logged through
// vine:core:redact without their original error messages.
func Render(value any, options ...Option) (Result, error) {
	return internalredact.Render(value, options...)
}
