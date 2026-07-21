package httputil

import (
	"net/http"
	"strings"
	"testing"
)

func TestCopyHeaderAndClearBodyHeaders(t *testing.T) {
	header := http.Header{}
	CopyHeader(header, http.Header{
		"X-Test":         []string{"one", "two"},
		"Content-Length": []string{"12"},
	})
	ClearBodyHeaders(header)

	if got := header.Values("X-Test"); strings.Join(got, ",") != "one,two" {
		t.Fatalf("unexpected copied header: %v", got)
	}
	if got := header.Get("Content-Length"); got != "" {
		t.Fatalf("unexpected content length: %s", got)
	}
}

func TestIsEventStreamResponse(t *testing.T) {
	for _, contentType := range []string{
		"text/event-stream; charset=utf-8",
		"Text/Event-Stream",
	} {
		response := &http.Response{
			Header: http.Header{"Content-Type": []string{contentType}},
		}

		if !IsEventStreamResponse(response) {
			t.Fatalf("expected event stream response: %s", contentType)
		}
	}
}

func TestIsEventStreamRequest(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "http://demo.local/events", nil)
	request.Header.Set("Accept", "Text/Event-Stream")

	if !IsEventStreamRequest(request) {
		t.Fatal("expected event stream request")
	}
}
