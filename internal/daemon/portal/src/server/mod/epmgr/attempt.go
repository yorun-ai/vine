package epmgr

import (
	"context"
	"errors"
	"io"
	"net"
	"syscall"
	"time"
)

const (
	transportFailureThreshold = 3
	initialCooldown           = 5 * time.Second
	maxCooldown               = 30 * time.Second
)

type _EndpointState struct {
	failures int
	cooldown time.Duration
	retryAt  time.Time
	probing  bool
	epoch    uint64
}

// Attempt owns a single endpoint selection, including the exclusive recovery
// probe after a cooldown. Results from retired registrations or earlier circuit
// states are ignored. All fields are protected by the manager mutex.
type Attempt struct {
	manager      *Manager
	route        *_Route
	registration any
	state        *_EndpointState
	epoch        uint64
	probe        bool
	finished     bool
}

// Finish reports transport completion once. A nil error means the response was
// received; business errors in that response do not make an endpoint unhealthy.
// Cancellation or an unclassified error releases a probe without marking it healthy.
func (a *Attempt) Finish(err error) {
	m := a.manager
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if a.finished {
		return
	}
	a.finished = true
	s := a.state
	if m.routesByPrefix[a.route.prefix] != a.route || a.route.endpointStates[a.registration] != s || s.epoch != a.epoch {
		return
	}
	if err == nil {
		s.failures = 0
		if a.probe {
			s.cooldown = 0
			s.retryAt = time.Time{}
			s.probing = false
			s.epoch++
		}
		return
	}
	failure, immediate := endpointFailure(err)
	if !failure {
		if a.probe {
			s.probing = false
			s.retryAt = time.Now().Add(s.cooldown)
			s.epoch++
		}
		return
	}
	s.failures++
	if !a.probe && !immediate && s.failures < transportFailureThreshold {
		return
	}
	if s.cooldown == 0 {
		s.cooldown = initialCooldown
	} else {
		s.cooldown = min(2*s.cooldown, maxCooldown)
	}
	s.retryAt = time.Now().Add(s.cooldown)
	s.probing = false
	s.epoch++
	epmgrLogger.Warn("rpc endpoint temporarily isolated", "registration", a.registration, "cooldown", s.cooldown, "error", err)
}

// TrackBody completes an attempt when the response body reaches EOF or fails.
// Closing an unread response abandons the attempt without reporting success.
func (a *Attempt) TrackBody(body io.ReadCloser) io.ReadCloser {
	return new(_ObservedBody{ReadCloser: body, attempt: a})
}

type _ObservedBody struct {
	io.ReadCloser
	attempt *Attempt
}

func (b *_ObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF {
		b.attempt.Finish(nil)
	} else if err != nil {
		b.attempt.Finish(err)
	}
	return n, err
}

func (b *_ObservedBody) Close() error {
	defer b.attempt.Finish(context.Canceled)
	return b.ReadCloser.Close()
}

func endpointFailure(err error) (bool, bool) {
	if errors.Is(err, context.Canceled) {
		return false, false
	}
	var op *net.OpError
	if (errors.As(err, &op) && op.Op == "dial") || errors.Is(err, syscall.ECONNREFUSED) {
		return true, true
	}
	// An overall RPC deadline does not identify a failed network endpoint.
	if errors.Is(err, context.DeadlineExceeded) {
		return false, false
	}
	var networkError net.Error
	if op != nil || (errors.As(err, &networkError) && networkError.Timeout()) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true, false
	}
	return false, false
}
