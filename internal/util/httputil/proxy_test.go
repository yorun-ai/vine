package httputil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"net/http/httptest"
	stdhttputil "net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServeReverseProxyKeepsResultRequestLocal(t *testing.T) {
	log, _ := newTestProxyLogger(t, "web:devproxy")
	proxy := &stdhttputil.ReverseProxy{
		Rewrite: func(r *stdhttputil.ProxyRequest) {},
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/failed" {
				return nil, errors.New("transport failed")
			}
			return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("upstream unavailable"))}, nil
		}),
	}
	var group sync.WaitGroup
	for i := range 16 {
		group.Go(func() {
			path, status, want := "/", 503, true
			if i%2 != 0 {
				path, status, want = "/failed", 502, false
			}
			request := httptest.NewRequestWithContext(t.Context(), "GET", "http://backend"+path, nil)
			response := httptest.NewRecorder()
			forwarded := ServeReverseProxy(proxy, response, request, log)
			if response.Code != status || forwarded != want {
				t.Errorf("%s: status=%d, forwarded=%v", path, response.Code, forwarded)
			}
			if request.Context() != t.Context() {
				t.Error("proxy changed the incoming context")
			}
		})
	}
	group.Wait()
	if proxy.ModifyResponse != nil || proxy.ErrorHandler != nil || proxy.ErrorLog != nil {
		t.Fatal("shared proxy configuration was modified")
	}
}

func TestServeReverseProxyPreservesLocalFailures(t *testing.T) {
	for _, stage := range []string{"transport", "modify", "body"} {
		t.Run(stage, func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			ctx := context.WithValue(t.Context(), http.ServerContextKey, new(http.Server))
			failure := errors.New("upstream failure")
			proxy := &stdhttputil.ReverseProxy{
				Rewrite: func(r *stdhttputil.ProxyRequest) {},
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if stage == "transport" {
						return nil, failure
					}
					var body io.ReadCloser = io.NopCloser(strings.NewReader("response"))
					if stage == "body" {
						body = io.NopCloser(proxyFailingReader{err: failure})
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
				}),
				ModifyResponse: func(r *http.Response) error {
					if stage == "modify" {
						return failure
					}
					return nil
				},
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) { w.WriteHeader(504) },
			}
			recorder := httptest.NewRecorder()
			var recovered any
			var forwarded bool
			func() {
				defer func() { recovered = recover() }()
				forwarded = ServeReverseProxy(proxy, recorder, httptest.NewRequestWithContext(ctx, "GET", "http://backend/", nil), log)
			}()
			if forwarded {
				t.Fatal("local failure retained upstream classification")
			}
			if stage == "body" {
				if recovered != http.ErrAbortHandler {
					t.Fatalf("panic=%v, want ErrAbortHandler", recovered)
				}
			} else if recovered != nil || recorder.Code != 504 {
				t.Fatalf("original error handler not preserved: status=%d panic=%v", recorder.Code, recovered)
			}
			record := readProxyLog(t, path)
			if record["level"] != "ERROR" || record["error"] != failure.Error() {
				t.Fatalf("failure log=%v", record)
			}
		})
	}
}

type proxyFailingReader struct{ err error }

func (r proxyFailingReader) Read([]byte) (int, error) { return 0, r.err }

func TestServeReverseProxyAbortsOnlyClientCancellation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel bool
		err    error
		abort  bool
	}{
		{"client cancellation", true, context.Canceled, true},
		{"EOF after cancellation", true, io.EOF, true},
		{"upstream cancellation", false, context.Canceled, false},
		{"deadline after cancellation", true, context.DeadlineExceeded, false},
		{"unrelated failure after cancellation", true, io.ErrClosedPipe, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			called := false
			proxy := &stdhttputil.ReverseProxy{
				Rewrite: func(r *stdhttputil.ProxyRequest) {},
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if tc.cancel {
						cancel()
					}
					return nil, tc.err
				}),
				ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
					called = true
					w.WriteHeader(http.StatusBadGateway)
				},
			}
			response := httptest.NewRecorder()
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				if ServeReverseProxy(proxy, response, httptest.NewRequestWithContext(ctx, "GET", "http://backend/", nil), log) {
					t.Error("failed request classified as forwarded")
				}
			}()
			level := "ERROR"
			if tc.abort {
				level = "DEBUG"
				if recovered != http.ErrAbortHandler || called || response.Code != http.StatusOK {
					t.Fatalf("canceled request: panic=%v errorHandler=%v status=%d", recovered, called, response.Code)
				}
			} else if recovered != nil || !called || response.Code != http.StatusBadGateway {
				t.Fatalf("failed request: panic=%v errorHandler=%v status=%d", recovered, called, response.Code)
			}
			record := readProxyLog(t, path)
			if record["level"] != level || record["error"] != tc.err.Error() {
				t.Fatalf("failure log=%v", record)
			}
		})
	}
}

func TestServeReverseProxyWriteFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cancel bool
		err    error
		level  string
	}{
		{"write failure", false, io.ErrClosedPipe, "ERROR"},
		{"canceled HTTP2 write", true, errors.New("http2: stream closed"), "DEBUG"},
		{"live HTTP2 write", false, errors.New("http2: stream closed"), "ERROR"},
		{"client cancellation", true, context.Canceled, "DEBUG"},
		{"upstream cancellation", false, context.Canceled, "ERROR"},
		{"deadline", true, context.DeadlineExceeded, "ERROR"},
		{"unrelated failure after cancellation", true, io.ErrClosedPipe, "ERROR"},
		{"short write after cancellation", true, nil, "ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx = context.WithValue(ctx, http.ServerContextKey, new(http.Server))
			proxy := &stdhttputil.ReverseProxy{
				Rewrite: func(r *stdhttputil.ProxyRequest) {},
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}, nil
				}),
			}
			writer := &proxyFailingWriter{ResponseRecorder: httptest.NewRecorder(), write: func([]byte) (int, error) {
				if tc.cancel {
					cancel()
				}
				return 0, tc.err
			}}
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				ServeReverseProxy(proxy, writer, httptest.NewRequestWithContext(ctx, "GET", "http://backend/", nil), log)
			}()
			if recovered != http.ErrAbortHandler {
				t.Fatalf("write failure: panic=%v", recovered)
			}
			want := tc.err
			if want == nil {
				want = io.ErrShortWrite
			}
			record := readProxyLog(t, path)
			if record["level"] != tc.level || record["error"] != want.Error() {
				t.Fatalf("write failure log=%v", record)
			}
		})
	}
}

func TestServeReverseProxyFlushFailures(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failure := io.ErrClosedPipe
			level := "ERROR"
			if canceled {
				failure, level = context.Canceled, "DEBUG"
			}
			proxy := &stdhttputil.ReverseProxy{
				Rewrite:       func(r *stdhttputil.ProxyRequest) {},
				FlushInterval: -1,
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}, nil
				}),
			}
			writer := &proxyFailingFlusher{ResponseRecorder: httptest.NewRecorder(), flush: func() error {
				if canceled {
					cancel()
				}
				return failure
			}}
			if ServeReverseProxy(proxy, writer, httptest.NewRequestWithContext(ctx, "GET", "http://backend/", nil), log) {
				t.Fatal("flush failure retained upstream classification")
			}
			record := readProxyLog(t, path)
			if record["level"] != level || record["error"] != failure.Error() {
				t.Fatalf("flush failure log=%v", record)
			}
		})
	}
}

func TestServeReverseProxyDoesNotDuplicateBodyFailureLogs(t *testing.T) {
	var standardOutput bytes.Buffer
	original := stdlog.Writer()
	stdlog.SetOutput(&standardOutput)
	t.Cleanup(func() { stdlog.SetOutput(original) })
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			proxy := &stdhttputil.ReverseProxy{
				Rewrite: func(r *stdhttputil.ProxyRequest) {},
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if canceled {
						cancel()
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(proxyFailingReader{err: io.ErrUnexpectedEOF})}, nil
				}),
			}
			// With no ServerContextKey, ReverseProxy also logs its suppressed panic.
			if ServeReverseProxy(proxy, httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, "GET", "http://backend/", nil), log) {
				t.Fatal("body failure retained upstream classification")
			}
			level := "ERROR"
			if canceled {
				level = "DEBUG"
			}
			record := readProxyLog(t, path)
			if record["level"] != level || record["error"] != io.ErrUnexpectedEOF.Error() {
				t.Fatalf("body failure log=%v", record)
			}
			if standardOutput.Len() != 0 {
				t.Fatalf("duplicate standard log: %s", standardOutput.String())
			}
		})
	}
}

func TestServeReverseProxyRejectsInvalidUpgrade(t *testing.T) {
	log, path := newTestProxyLogger(t, "web:devproxy")
	proxy := &stdhttputil.ReverseProxy{
		Rewrite: func(r *stdhttputil.ProxyRequest) {},
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 101, Header: http.Header{"Connection": {"Upgrade"}, "Upgrade": {"other"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}),
	}
	request := httptest.NewRequestWithContext(t.Context(), "GET", "http://backend/hmr", nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	response := httptest.NewRecorder()
	forwarded := ServeReverseProxy(proxy, response, request, log)
	if response.Code != 502 || forwarded {
		t.Fatalf("invalid upgrade: status=%d, forwarded=%v", response.Code, forwarded)
	}
	record := readProxyLog(t, path)
	if record["level"] != "ERROR" || !strings.Contains(record["error"].(string), "backend tried to switch protocol") {
		t.Fatalf("upgrade failure log=%v", record)
	}
}

func TestServeReverseProxyClientDisconnect(t *testing.T) {
	for _, stage := range []string{"streaming body", "pending upgrade"} {
		t.Run(stage, func(t *testing.T) {
			log, path := newTestProxyLogger(t, "web:devproxy")
			backendEntered := make(chan struct{})
			backendCanceled := make(chan error, 1)
			backend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stage == "streaming body" {
					_, _ = io.WriteString(w, "partial")
					_ = http.NewResponseController(w).Flush()
				}
				close(backendEntered)
				select {
				case <-r.Context().Done():
					backendCanceled <- r.Context().Err()
				case <-t.Context().Done():
				}
			}))
			backend.Start()
			target, err := url.Parse(backend.URL)
			if err != nil {
				t.Fatal(err)
			}
			proxy := stdhttputil.NewSingleHostReverseProxy(target)
			proxy.Transport = backend.Client().Transport
			finished := make(chan bool, 1)
			frontend := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var forwarded bool
				defer func() { finished <- forwarded }()
				forwarded = ServeReverseProxy(proxy, w, r, log)
			}))
			frontend.Start()
			connection, err := net.DialTimeout("tcp", strings.TrimPrefix(frontend.URL, "http://"), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = connection.Close() })
			if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			headers := ""
			if stage == "pending upgrade" {
				headers = "Connection: Upgrade\r\nUpgrade: websocket\r\n"
			}
			if _, err := fmt.Fprintf(connection, "GET /hmr HTTP/1.1\r\nHost: demo.local\r\n%s\r\n", headers); err != nil {
				t.Fatal(err)
			}
			select {
			case <-backendEntered:
			case <-time.After(5 * time.Second):
				t.Fatal("backend did not receive request")
			}
			if stage == "streaming body" {
				response, err := http.ReadResponse(bufio.NewReader(connection), nil)
				if err != nil {
					t.Fatal(err)
				}
				body := make([]byte, len("partial"))
				if _, err := io.ReadFull(response.Body, body); err != nil || string(body) != "partial" {
					t.Fatalf("stream did not flush: body=%q error=%v", body, err)
				}
			}
			if err := connection.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-backendCanceled:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("backend cancellation=%v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("client disconnect did not cancel backend request")
			}
			select {
			case forwarded := <-finished:
				if forwarded {
					t.Fatal("canceled transfer retained upstream classification")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("proxy did not exit after client disconnect")
			}
			record := readProxyLog(t, path)
			if record["level"] != "DEBUG" || record["msg"] != "web proxy canceled" {
				t.Fatalf("client disconnect log=%v", record)
			}
		})
	}
}

type proxyFailingWriter struct {
	*httptest.ResponseRecorder
	write func([]byte) (int, error)
}

func (w *proxyFailingWriter) Write(buffer []byte) (int, error) { return w.write(buffer) }

type proxyFailingFlusher struct {
	*httptest.ResponseRecorder
	flush func() error
}

func (w *proxyFailingFlusher) FlushError() error { return w.flush() }

func BenchmarkServeReverseProxy(b *testing.B) {
	log, _ := newTestProxyLogger(b, "benchmark:proxy")
	for _, size := range []int{1024, 64 * 1024, 1024 * 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			payload := bytes.Repeat([]byte("a"), size)
			proxy := &stdhttputil.ReverseProxy{
				Rewrite: func(r *stdhttputil.ProxyRequest) {},
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: int64(size), Body: io.NopCloser(bytes.NewReader(payload))}, nil
				}),
			}
			request := httptest.NewRequestWithContext(b.Context(), "GET", "http://backend/", nil)
			for _, variant := range []string{"standard", "wrapped"} {
				b.Run(variant, func(b *testing.B) {
					writer := &proxyDiscardWriter{header: make(http.Header)}
					b.ReportAllocs()
					b.SetBytes(int64(size))
					for b.Loop() {
						if variant == "standard" {
							proxy.ServeHTTP(writer, request)
						} else {
							if !ServeReverseProxy(proxy, writer, request, log) {
								b.Fatal("response was not forwarded")
							}
						}
					}
				})
			}
		})
	}
}

type proxyDiscardWriter struct{ header http.Header }

func (w *proxyDiscardWriter) Header() http.Header              { return w.header }
func (w *proxyDiscardWriter) WriteHeader(int)                  {}
func (w *proxyDiscardWriter) Write(buffer []byte) (int, error) { return len(buffer), nil }
