package httputil

import (
	"errors"
	"io"
	stdlog "log"
	"net/http"
	stdhttputil "net/http/httputil"
	"sync/atomic"

	"go.yorun.ai/vine/internal/core/logger"
)

var discardProxyErrorLog = stdlog.New(io.Discard, "", 0)

// ServeReverseProxy reports whether an upstream response was forwarded without
// a local failure. It logs failures and aborts canceled requests with ErrAbortHandler.
// Copying the configuration keeps callbacks request-local while retaining the
// shared transport, buffer pool and existing response/error hooks. Failure logs
// belong to this wrapper, replacing ReverseProxy.ErrorLog to avoid duplicates.
func ServeReverseProxy(proxy *stdhttputil.ReverseProxy, w http.ResponseWriter, r *http.Request, log *logger.Logger) bool {
	transfer := new(_ProxyTransfer{ResponseWriter: w, proxy: proxy, request: r, log: log})
	next := *proxy
	next.ErrorLog = discardProxyErrorLog
	next.ModifyResponse = transfer.modifyResponse
	next.ErrorHandler = transfer.handleError
	next.ServeHTTP(transfer, r)
	return !transfer.failed.Load()
}

type _ProxyTransfer struct {
	http.ResponseWriter
	proxy   *stdhttputil.ReverseProxy
	request *http.Request
	log     *logger.Logger
	body    io.ReadCloser
	failed  atomic.Bool
}

func (t *_ProxyTransfer) fail(err error) {
	// Timer-driven flushes can fail while the body is being read. Record the first
	// failure once, including its original cause before ErrAbortHandler replaces it.
	if t.failed.CompareAndSwap(false, true) {
		logProxyFailure(t.request.Context(), t.log, "web proxy", err)
	}
}

func (t *_ProxyTransfer) modifyResponse(response *http.Response) error {
	if t.proxy.ModifyResponse != nil {
		if err := t.proxy.ModifyResponse(response); err != nil {
			return err
		}
	}
	// ReverseProxy owns upgraded streams; preserve their io.ReadWriteCloser.
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.body = response.Body
		response.Body = t
	}
	return nil
}

func (t *_ProxyTransfer) handleError(w http.ResponseWriter, request *http.Request, err error) {
	t.fail(err)
	// The client is gone. Do not turn its cancellation into a local 502 and an
	// Error completion log; let the HTTP transport terminate the request.
	if IsCanceledResponseTransfer(t.request.Context(), err) {
		panic(http.ErrAbortHandler)
	}
	if t.proxy.ErrorHandler != nil {
		t.proxy.ErrorHandler(w, request, err)
	} else {
		w.WriteHeader(http.StatusBadGateway)
	}
}

func (t *_ProxyTransfer) Write(buffer []byte) (int, error) {
	n, err := t.ResponseWriter.Write(buffer)
	if err == nil && n != len(buffer) {
		err = io.ErrShortWrite
	}
	if err != nil {
		t.fail(err)
	}
	return n, err
}

func (t *_ProxyTransfer) FlushError() error {
	err := http.NewResponseController(t.ResponseWriter).Flush()
	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		t.fail(err)
	}
	return err
}

// Unwrap preserves ResponseController capabilities, including WebSocket hijacking.
func (t *_ProxyTransfer) Unwrap() http.ResponseWriter { return t.ResponseWriter }

func (t *_ProxyTransfer) Read(buffer []byte) (int, error) {
	n, err := t.body.Read(buffer)
	if err != nil && err != io.EOF {
		t.fail(err)
	}
	return n, err
}

func (t *_ProxyTransfer) Close() error { return t.body.Close() }
