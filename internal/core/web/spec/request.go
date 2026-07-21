package spec

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.yorun.ai/vine/core/meta"
	"go.yorun.ai/vine/util/vstring"
)

const (
	HeaderWebTrace     = "vweb-trace"
	HeaderWebActor     = "vweb-actor"
	HeaderWebInitiator = "vweb-initiator"
	HeaderWebOptions   = "vweb-options"
)

type RequestMeta struct {
	Trace     meta.Trace
	Initiator meta.Initiator
	Actor     meta.Actor
}

type Options struct {
	Timeout time.Duration
}

func DecodeOptionsFromHeader(header http.Header) (*Options, error) {
	value := header.Get(HeaderWebOptions)
	if value == "" {
		return &Options{}, nil
	}

	fields, err := vstring.DecodeDelimited(value)
	if err != nil {
		return nil, fmt.Errorf("invalid header %s", HeaderWebOptions)
	}

	timeoutValue := fields["timeout"]
	if timeoutValue == "" || len(fields) != 1 {
		return nil, fmt.Errorf("invalid header %s", HeaderWebOptions)
	}

	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil || timeout <= 0 {
		return nil, fmt.Errorf("invalid header %s", HeaderWebOptions)
	}

	return &Options{Timeout: timeout}, nil
}

func EncodeOptionsToHeader(header http.Header, options *Options) {
	if options == nil || options.Timeout <= 0 {
		return
	}
	header.Set(HeaderWebOptions, vstring.EncodeDelimited("timeout", options.Timeout.String()))
}

func EncodeRequestOptionsToHeader(header http.Header, ctx context.Context) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return
	}
	EncodeOptionsToHeader(header, &Options{Timeout: timeout})
}

func DecodeTraceFromHeader(header http.Header) (meta.Trace, error) {
	value := header.Get(HeaderWebTrace)
	if value == "" {
		return nil, fmt.Errorf("header vweb-trace not found")
	}

	trace, err := meta.DecodeTraceFromDelimited(value)
	if err != nil {
		return nil, fmt.Errorf("invalid header vweb-trace")
	}

	return trace, nil
}

func EncodeTraceToHeader(header http.Header, trace meta.Trace) {
	header.Set(HeaderWebTrace, meta.EncodeTraceToDelimited(trace))
}

func decodeActorFromHeader(header http.Header) (meta.Actor, error) {
	value := header.Get(HeaderWebActor)
	if value == "" {
		return nil, fmt.Errorf("header vweb-actor not found")
	}

	actor, err := meta.DecodeActorFromBase64(value)
	if err != nil {
		return nil, fmt.Errorf("decode vweb-actor failed: %s", err)
	}
	return actor, nil
}

func decodeInitiatorFromHeader(header http.Header) (meta.Initiator, error) {
	value := header.Get(HeaderWebInitiator)
	if value == "" {
		return nil, fmt.Errorf("header vweb-initiator not found")
	}

	initiator, err := meta.DecodeInitiatorFromBase64(value)
	if err != nil {
		return nil, fmt.Errorf("decode vweb-initiator failed: %s", err)
	}
	return initiator, nil
}

func encodeInitiatorToHeader(header http.Header, metaInitiator meta.Initiator) {
	header.Set(HeaderWebInitiator, meta.EncodeInitiatorToBase64(metaInitiator))
}

func DecodeRequestMeta(header http.Header) (*RequestMeta, error) {
	trace, err := DecodeTraceFromHeader(header)
	if err != nil {
		return nil, err
	}

	initiator, err := decodeInitiatorFromHeader(header)
	if err != nil {
		return nil, err
	}

	actor, err := decodeActorFromHeader(header)
	if err != nil {
		return nil, err
	}

	return &RequestMeta{
		Trace:     trace,
		Initiator: initiator,
		Actor:     actor,
	}, nil
}
