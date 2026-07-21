package meta

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"

	"go.yorun.ai/vine/util/vstring"
)

type Trace interface {
	Id() string
	Span() string
	ParentSpan() string
	NewChildTrace() Trace
}

type _Trace struct {
	id         string
	parentSpan string
	span       string
}

func NewId() string {
	return newId()
}

func NewSpan() string {
	return newSpan()
}

func IsValidSpan(span string) bool {
	return checkSpanFormat(span)
}

func NewTrace(id string, span string) (Trace, error) {
	if !IsValidId(id) {
		return nil, fmt.Errorf("invalid id format")
	}

	if span == "" {
		span = newSpan()
	}
	if !checkSpanFormat(span) {
		return nil, fmt.Errorf("invalid span format")
	}

	return &_Trace{
		id:   id,
		span: span,
	}, nil
}

func DecodeTraceFromDelimited(value string) (Trace, error) {
	fields, err := vstring.DecodeDelimited(value)
	if err != nil {
		return nil, err
	}

	id := fields["id"]
	span := fields["span"]
	if id == "" || span == "" {
		return nil, fmt.Errorf("missing trace field")
	}

	return NewTrace(id, span)
}

func DecodeTraceFromDelimitedOrNewSpan(value string) (Trace, error) {
	fields, err := vstring.DecodeDelimited(value)
	if err != nil {
		return nil, err
	}

	id := fields["id"]
	if id == "" {
		return nil, fmt.Errorf("missing trace id")
	}

	span := fields["span"]
	if span == "" {
		span = NewSpan()
	}

	return NewTrace(id, span)
}

func EncodeTraceToDelimited(trace Trace) string {
	return vstring.EncodeDelimited(
		"id", trace.Id(),
		"span", trace.Span(),
	)
}

func (t *_Trace) Id() string {
	return t.id
}

func (t *_Trace) Span() string {
	return t.span
}

func (t *_Trace) ParentSpan() string {
	return t.parentSpan
}

func (t *_Trace) NewChildTrace() Trace {
	return &_Trace{
		id:         t.id,
		parentSpan: t.span,
		span:       newSpan(),
	}
}

func InitialTrace() Trace {
	return &_Trace{
		id:   newId(),
		span: newSpan(),
	}
}

const (
	traceIDLength = 16
	traceIDZeros  = "00000000000000000000000000000000"
	spanIDLength  = 8
	spanIDZeros   = "0000000000000000"
)

func newId() string {
	for {
		var raw [traceIDLength]byte
		_, err := rand.Read(raw[:])
		if err != nil {
			panic(err)
		}

		id := hex.EncodeToString(raw[:])
		if id != traceIDZeros {
			return id
		}
	}
}

var idRegx = regexp.MustCompile(`^[0-9a-f]{32}$`)

func IsValidId(id string) bool {
	return idRegx.MatchString(id) && id != traceIDZeros
}

func newSpan() string {
	for {
		var raw [spanIDLength]byte
		_, err := rand.Read(raw[:])
		if err != nil {
			panic(err)
		}

		span := hex.EncodeToString(raw[:])
		if span != spanIDZeros {
			return span
		}
	}
}

var spanRegx = regexp.MustCompile(`^[0-9a-f]{16}$`)

func checkSpanFormat(span string) bool {
	return spanRegx.MatchString(span) && span != spanIDZeros
}
