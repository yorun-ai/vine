package webgw

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/util/vcode"
)

func TestForwardResponseIntegrity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		length      string
		body        string
		abort       bool
		compress    bool
		method      string
		status      int
		wantFailure bool
	}{
		{name: "complete", length: "7", body: "package", status: 200},
		{name: "unknown", body: "package", status: 200},
		{name: "truncated", length: "70010", body: strings.Repeat("partial", 10000), status: 200, wantFailure: true},
		{name: "aborted unknown length", body: strings.Repeat("partial", 10000), abort: true, status: 200, wantFailure: true},
		{name: "compressed truncation", length: "10", body: "partial", compress: true, status: 200, wantFailure: true},
		{name: "compressed", length: "5000", body: strings.Repeat("x", 5000), compress: true, status: 200},
		{name: "head", method: "HEAD", length: "100", status: 200},
		{name: "not modified", length: "100", status: 304},
		{name: "no content", status: 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := "link+inproc://response-integrity"
			// Model the intermediate hop copying an application's response, including aborts.
			app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				if tc.compress {
					w.Header().Set("Content-Type", "text/plain")
				}
				w.WriteHeader(tc.status)
				if tc.body != "" {
					_, _ = w.Write([]byte(tc.body))
				}
				if tc.abort {
					panic(http.ErrAbortHandler)
				}
			})
			ingressinproc.Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response, err := httputil.InprocRoundTrip(app, r)
				if err != nil {
					http.Error(w, "upstream failed", 502)
					return
				}
				defer response.Body.Close()
				httputil.CopyHeader(w.Header(), response.Header)
				w.WriteHeader(response.StatusCode)
				if _, err := httputil.CopyResponseBody(w, response); err != nil {
					panic(http.ErrAbortHandler)
				}
			}))
			t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
			gateway := newTestWebGateway(t, map[string]string{
				watched.FormatWebRegistrationKey("admin@demo.app", "demo.app", "instance-1"): vcode.MustMarshalJsonS(watched.WebRegistration{
					Endpoint: endpoint + "/web/proxy/in/instance-1/admin@demo.app", WebSkelName: "admin@demo.app", AppName: "demo.app", AppInstanceId: "instance-1",
				}),
			})
			for _, h2 := range []bool{false, true} {
				downstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { gateway.Serve(testContext(w, r)) }))
				downstream.EnableHTTP2 = h2
				downstream.StartTLS()
				method := tc.method
				if method == "" {
					method = "GET"
				}
				req, _ := http.NewRequest(method, downstream.URL, nil)
				if tc.compress {
					req.Header.Set("Accept-Encoding", "gzip")
				}
				response, err := downstream.Client().Do(req)
				var body []byte
				if err == nil {
					body, err = io.ReadAll(response.Body)
					response.Body.Close()
				}
				downstream.Close()
				if tc.wantFailure {
					if !tc.compress && (response == nil || response.StatusCode != http.StatusOK) {
						t.Fatalf("expected committed 200 before interruption, response=%v err=%v", response, err)
					}
					if err == nil && response.StatusCode < 500 {
						t.Fatalf("http2=%v: truncated transfer succeeded: %q", h2, body)
					}
					if tc.compress && err == nil && response.StatusCode != 502 {
						t.Fatalf("status=%d", response.StatusCode)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if !tc.compress && tc.status != 304 && tc.length != "" && response.Header.Get("Content-Length") != tc.length {
					t.Fatalf("length lost: %v", response.Header)
				}
				if !tc.compress && string(body) != tc.body {
					t.Fatalf("body=%q", body)
				}
				if tc.compress {
					if response.Header.Get("Content-Encoding") != "gzip" {
						t.Fatalf("encoding: %v", response.Header)
					}
					reader, err := gzip.NewReader(bytes.NewReader(body))
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := io.ReadAll(reader)
					reader.Close()
					if err != nil || string(decoded) != tc.body {
						t.Fatalf("compressed body corrupted: %v", err)
					}
					if response.Header.Get("Content-Length") == tc.length {
						t.Fatal("original content length retained after compression")
					}
				}
			}
		})
	}
}
