package redact

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"go.yorun.ai/vine/internal/core/logger"
)

const (
	defaultMaxDepth           = 16
	defaultMaxNodes           = 10_000
	defaultMaxCollectionItems = 100
	defaultMaxStringBytes     = 4 * 1024
	defaultMaxOutputBytes     = 64 * 1024
)

var failureLogger = logger.New("vine:core:redact")

// Limits bound the work and output produced while rendering a value.
type Limits struct {
	MaxDepth           int
	MaxNodes           int
	MaxCollectionItems int
	MaxStringBytes     int
	MaxOutputBytes     int
}

// DefaultLimits returns the limits used when Option.Limits is nil.
func DefaultLimits() Limits {
	return Limits{
		MaxDepth:           defaultMaxDepth,
		MaxNodes:           defaultMaxNodes,
		MaxCollectionItems: defaultMaxCollectionItems,
		MaxStringBytes:     defaultMaxStringBytes,
		MaxOutputBytes:     defaultMaxOutputBytes,
	}
}

// Option controls how a value is rendered.
type Option struct {
	// RevealSensitive disables explicit sensitive-value and field masking.
	// Binary values are still replaced with summaries.
	RevealSensitive bool
	// RootSensitive treats the root value as sensitive as a whole.
	RootSensitive bool
	// Sanitizer optionally creates a safe projection before field redaction.
	// It is skipped when RootSensitive masks the root value.
	Sanitizer func(any) (any, error)
	// Limits overrides the default traversal and output limits. Every field
	// must be greater than zero.
	Limits *Limits
}

// Result is the JSON-safe rendering of a value.
type Result struct {
	// JSON contains the rendered value without a trailing newline.
	JSON string
	// Redacted reports whether at least one sensitive or binary value was replaced.
	Redacted bool
	// Truncated reports whether at least one value or the final JSON output was
	// replaced because a rendering limit was reached.
	Truncated bool
}

type _Failure struct {
	kind  string
	cause error
}

func (e *_Failure) Error() string {
	return fmt.Sprintf("redact %s failed: %v", e.kind, e.cause)
}

func (e *_Failure) Unwrap() error {
	return e.cause
}

func newFailure(kind string, cause error) error {
	if _, ok := errors.AsType[*_Failure](cause); ok {
		return cause
	}
	return &_Failure{kind: kind, cause: cause}
}

func failureKind(err error) string {
	if failure, ok := errors.AsType[*_Failure](err); ok {
		return failure.kind
	}
	return "unknown"
}

func reportFailure(err error) {
	if err == nil {
		return
	}
	failureLogger.Error(
		"value redaction failed",
		"failureKind", failureKind(err),
	)
}

// Render converts value to JSON while masking explicitly sensitive values and fields.
// Failures are logged without their potentially sensitive error messages.
func Render(value any, options ...Option) (result Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = Result{}
			err = newFailure("panic", fmt.Errorf("render redacted value: %v", recovered))
		}
		reportFailure(err)
	}()
	if len(options) > 1 {
		return Result{}, newFailure("invalid_option", fmt.Errorf("redact.Render accepts at most one Option"))
	}
	option := Option{}
	if len(options) > 0 {
		option = options[0]
	}
	limits := DefaultLimits()
	if option.Limits != nil {
		limits = *option.Limits
	}
	if err := validateLimits(limits); err != nil {
		return Result{}, newFailure("invalid_limits", err)
	}
	if option.RootSensitive && !option.RevealSensitive {
		return Result{JSON: `"<redacted>"`, Redacted: true}, nil
	}
	if option.Sanitizer != nil {
		value, err = option.Sanitizer(value)
		if err != nil {
			return Result{}, newFailure("sanitize", err)
		}
	}
	state := newProjectionState(option, limits)
	projected, err := state.project(reflect.ValueOf(value), 0)
	if err != nil {
		return Result{}, newFailure("project", err)
	}
	encoded := &_LimitedBuffer{limit: limits.MaxOutputBytes}
	if err := json.MarshalWrite(
		encoded,
		projected,
		json.Deterministic(true),
		jsontext.EscapeForHTML(false),
	); err != nil {
		return Result{}, newFailure("encode", err)
	}
	rendered := encoded.String()
	if encoded.exceeded {
		state.truncated = true
		rendered = strconv.Quote(fmt.Sprintf("<truncated:json bytes=%d>", encoded.bytes))
	}
	return Result{
		JSON:      rendered,
		Redacted:  state.redacted,
		Truncated: state.truncated,
	}, nil
}

func validateLimits(limits Limits) error {
	if limits.MaxDepth <= 0 {
		return fmt.Errorf("redact max depth must be greater than zero")
	}
	if limits.MaxNodes <= 0 {
		return fmt.Errorf("redact max nodes must be greater than zero")
	}
	if limits.MaxCollectionItems <= 0 {
		return fmt.Errorf("redact max collection items must be greater than zero")
	}
	if limits.MaxStringBytes <= 0 {
		return fmt.Errorf("redact max string bytes must be greater than zero")
	}
	if limits.MaxOutputBytes <= 0 {
		return fmt.Errorf("redact max output bytes must be greater than zero")
	}
	return nil
}

type _LimitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	bytes    int
	exceeded bool
}

func (b *_LimitedBuffer) Write(value []byte) (int, error) {
	b.bytes += len(value)
	if !b.exceeded && b.buffer.Len()+len(value) <= b.limit {
		_, _ = b.buffer.Write(value)
	} else {
		b.exceeded = true
		b.buffer.Reset()
	}
	return len(value), nil
}

func (b *_LimitedBuffer) String() string {
	return b.buffer.String()
}
