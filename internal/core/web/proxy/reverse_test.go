package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestReverseProxyServeForwardsRequestPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotPath string
	var gotQuery string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte("proxied"))
	}))
	t.Cleanup(target.Close)

	reverseProxy := NewReverseProxy(Option{
		Target:            mustParseTestURL(t, target.URL),
		DialTimeout:       time.Second,
		DetectionInterval: time.Hour,
	})
	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/original?v=1", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}

	ok := reverseProxy.Serve(ginCtx)

	if !ok {
		t.Fatal("expected proxy to serve request")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != "proxied" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if gotPath != "/assets/app.js" {
		t.Fatalf("unexpected target path: %s", gotPath)
	}
	if gotQuery != "v=1" {
		t.Fatalf("unexpected target query: %s", gotQuery)
	}
}

func TestReverseProxyServeReturnsFalseWhenTargetUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reverseProxy := NewReverseProxy(Option{
		Target:            mustParseTestURL(t, "http://127.0.0.1:1"),
		DialTimeout:       10 * time.Millisecond,
		DetectionInterval: time.Hour,
	})
	recorder := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "http://demo.local/original", nil)
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}

	ok := reverseProxy.Serve(ginCtx)

	if ok {
		t.Fatal("expected proxy to skip unavailable target")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
}

func (*closeNotifyRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

func mustParseTestURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return parsed
}
