package web_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/core/web"
)

type proxyTransport func(*http.Request) (*http.Response, error)

func (f proxyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestServeReverseProxyPreservesCustomHooks(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(originalMode) })
	for _, failed := range []bool{false, true} {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest("GET", "/preview", nil)
		modified, handled := false, false
		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.Out.URL.Scheme = "http"
				r.Out.URL.Host = "upstream"
			},
			Transport: proxyTransport(func(r *http.Request) (*http.Response, error) {
				if failed {
					return nil, errors.New("connection failed")
				}
				return &http.Response{StatusCode: 502, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("upstream error")), ContentLength: 14}, nil
			}),
			ModifyResponse: func(r *http.Response) error {
				modified = true
				r.Header.Set("X-Proxy-Test", "kept")
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				handled = true
				http.Error(w, "local error", 503)
			},
		}
		web.ServeReverseProxy(proxy, ctx, logger.New("test:proxy"))
		if failed {
			if !handled || modified || w.Code != 503 {
				t.Fatalf("local failure: handled=%v modified=%v status=%d", handled, modified, w.Code)
			}
		} else if handled || !modified || w.Code != 502 || w.Header().Get("X-Proxy-Test") != "kept" || w.Body.String() != "upstream error" {
			t.Fatalf("upstream response: handled=%v modified=%v status=%d body=%s", handled, modified, w.Code, w.Body.String())
		}
	}
}
