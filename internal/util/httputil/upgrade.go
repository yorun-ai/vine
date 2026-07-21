package httputil

import (
	"io"
	"net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

func IsUpgradeRequest(r *http.Request) bool {
	return headerContainsToken(r.Header, "Connection", "upgrade") && r.Header.Get("Upgrade") != ""
}

func ForwardUpgrade(w http.ResponseWriter, r *http.Request, targetURL string, transport http.RoundTripper) error {
	target, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	proxy := &stdhttputil.ReverseProxy{
		Rewrite: func(request *stdhttputil.ProxyRequest) {
			request.Out.URL.Scheme = target.Scheme
			request.Out.URL.Host = target.Host
			request.Out.URL.Path = target.Path
			request.Out.URL.RawPath = target.RawPath
			request.Out.URL.RawQuery = target.RawQuery
			request.Out.Host = target.Host
			request.SetXForwarded()
		},
		Transport: NewUpgradeIdleTransport(transport, DefaultStreamIdleTimeout),
	}
	proxy.ServeHTTP(w, r)
	return nil
}

// NewUpgradeIdleTransport wraps upgraded response streams with a traffic-aware idle timeout.
func NewUpgradeIdleTransport(transport http.RoundTripper, idleTimeout time.Duration) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if idleTimeout <= 0 {
		idleTimeout = DefaultStreamIdleTimeout
	}
	return &_UpgradeIdleTransport{
		transport:   transport,
		idleTimeout: idleTimeout,
	}
}

type _UpgradeIdleTransport struct {
	transport   http.RoundTripper
	idleTimeout time.Duration
}

func (t *_UpgradeIdleTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.transport.RoundTrip(request)
	if err != nil || response.StatusCode != http.StatusSwitchingProtocols {
		return response, err
	}

	stream, ok := response.Body.(io.ReadWriteCloser)
	if !ok {
		return response, nil
	}
	response.Body = newUpgradeIdleReadWriteCloser(stream, t.idleTimeout)
	return response, nil
}

type _UpgradeIdleReadWriteCloser struct {
	io.ReadWriteCloser

	idleTimeout time.Duration
	mutex       sync.Mutex
	timer       *time.Timer
	done        chan struct{}
	closed      bool
	closeOnce   sync.Once
}

func newUpgradeIdleReadWriteCloser(stream io.ReadWriteCloser, idleTimeout time.Duration) *_UpgradeIdleReadWriteCloser {
	idleStream := &_UpgradeIdleReadWriteCloser{
		ReadWriteCloser: stream,
		idleTimeout:     idleTimeout,
		done:            make(chan struct{}),
	}
	idleStream.timer = time.NewTimer(idleTimeout)
	go func() {
		select {
		case <-idleStream.timer.C:
			_ = idleStream.Close()
		case <-idleStream.done:
		}
	}()
	return idleStream
}

func (c *_UpgradeIdleReadWriteCloser) Read(buffer []byte) (int, error) {
	n, err := c.ReadWriteCloser.Read(buffer)
	if n > 0 {
		c.touch()
	}
	return n, err
}

func (c *_UpgradeIdleReadWriteCloser) Write(buffer []byte) (int, error) {
	n, err := c.ReadWriteCloser.Write(buffer)
	if n > 0 {
		c.touch()
	}
	return n, err
}

func (c *_UpgradeIdleReadWriteCloser) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.mutex.Lock()
		c.closed = true
		c.timer.Stop()
		close(c.done)
		c.mutex.Unlock()
		err = c.ReadWriteCloser.Close()
	})
	return err
}

func (c *_UpgradeIdleReadWriteCloser) touch() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.closed {
		c.timer.Reset(c.idleTimeout)
	}
}

func headerContainsToken(header http.Header, key string, token string) bool {
	for _, value := range header.Values(key) {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
