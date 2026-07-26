package redact

import internalredact "go.yorun.ai/vine/internal/core/redact"

// Option controls how a value is rendered.
type Option = internalredact.Option

// Result is the JSON-safe rendering of a value.
type Result = internalredact.Result

// Render converts value to JSON while masking fields tagged
// skel:"sensitive" and commonly sensitive key names. Binary values are always
// replaced with a length and SHA-256 summary, including when RevealSensitive
// is enabled.
func Render(value any, options ...Option) (Result, error) {
	return internalredact.Render(value, options...)
}
