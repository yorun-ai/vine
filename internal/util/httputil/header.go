package httputil

import (
	"mime"
	"net/http"
	"strings"
)

func CopyHeader(dst http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func ClearBodyHeaders(header http.Header) {
	header.Del("Content-Length")
}

func IsEventStreamRequest(r *http.Request) bool {
	return isEventStreamMediaType(r.Header.Get("Accept"))
}

func IsEventStreamResponse(response *http.Response) bool {
	return isEventStreamMediaType(response.Header.Get("Content-Type"))
}

func isEventStreamMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.EqualFold(mediaType, "text/event-stream")
	}
	return strings.Contains(strings.ToLower(value), "text/event-stream")
}
