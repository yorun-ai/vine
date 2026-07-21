package site

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
)

func TestRedirectionSiteServesPermanentRedirect(t *testing.T) {
	site := NewRedirectionSite(true, "https://demo.local/new")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/old", nil)
	site.Serve(redirectionContext(recorder, request))

	assert.Equal(t, redirectionSiteName, site.Name())
	assert.Equal(t, http.StatusPermanentRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/new", recorder.Header().Get("Location"))
}

func TestRedirectionSiteServesTemporaryRedirect(t *testing.T) {
	site := NewRedirectionSite(false, "https://demo.local/later")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/old", nil)
	site.Serve(redirectionContext(recorder, request))

	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "https://demo.local/later", recorder.Header().Get("Location"))
}

func TestRedirectionSiteReplacesCaddyStylePlaceholders(t *testing.T) {
	site := NewRedirectionSite(false, "https://{host}/mirror{uri}?from={scheme}&method={method}")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://demo.local/old/page?debug=1", nil)
	site.Serve(redirectionContext(recorder, request))

	assert.Equal(t, "https://demo.local/mirror/old/page?debug=1?from=http&method=POST", recorder.Header().Get("Location"))
}

func TestRedirectionSiteKeepsUnknownPlaceholders(t *testing.T) {
	site := NewRedirectionSite(false, "https://demo.local/{unknown}")

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://demo.local/old", nil)
	site.Serve(redirectionContext(recorder, request))

	assert.Equal(t, "https://demo.local/{unknown}", recorder.Header().Get("Location"))
}

func redirectionContext(recorder http.ResponseWriter, request *http.Request) *spec.Context {
	return &spec.Context{
		Request:        request,
		ResponseWriter: recorder,
		RemoteAddr:     request.RemoteAddr,
	}
}
