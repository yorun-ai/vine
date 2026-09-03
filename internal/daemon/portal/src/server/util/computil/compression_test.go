package computil

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestPreferredEncodingPrefersZstd(t *testing.T) {
	if got := PreferredEncoding("gzip, zstd"); got != EncodingZstd {
		t.Fatalf("PreferredEncoding() = %q, want zstd", got)
	}
}

func TestCompressResponseBodySkipsSmallBody(t *testing.T) {
	body := []byte(strings.Repeat("a", CompressionThreshold))
	compressed, encoding := CompressResponseBody(body, "gzip, zstd")
	if encoding != "" {
		t.Fatalf("encoding = %q, want empty", encoding)
	}
	if string(compressed) != string(body) {
		t.Fatalf("unexpected body")
	}
}

func BenchmarkCompressResponseBody(b *testing.B) {
	for _, size := range []int{8 << 10, 64 << 10} {
		body := bytes.Repeat([]byte(`{"id":"123e4567-e89b-12d3-a456-426614174000","message":"vine benchmark"}`), size/72+1)
		body = body[:size]
		for _, encoding := range []string{EncodingGzip, EncodingZstd} {
			b.Run(fmt.Sprintf("%s/%dKiB", encoding, size>>10), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(body)))
				for b.Loop() {
					compressed, gotEncoding := CompressResponseBody(body, encoding)
					if gotEncoding != encoding || len(compressed) == 0 {
						b.Fatalf("CompressResponseBody() encoding = %q, bytes = %d", gotEncoding, len(compressed))
					}
				}
			})
		}
	}
}

func TestShouldCompressWebResponse(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "http://demo.local/app.js", nil)
	request.Header.Set("Accept-Encoding", "gzip, zstd")
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/javascript; charset=utf-8"},
		},
	}

	if !ShouldCompressWebResponse(request, response) {
		t.Fatal("ShouldCompressWebResponse() = false, want true")
	}
}

func TestShouldCompressWebResponseRejectsUnsafeCases(t *testing.T) {
	for name, configure := range map[string]func(*http.Request, *http.Response){
		"head": func(request *http.Request, response *http.Response) {
			request.Method = http.MethodHead
		},
		"encoded": func(request *http.Request, response *http.Response) {
			response.Header.Set("Content-Encoding", "gzip")
		},
		"range": func(request *http.Request, response *http.Response) {
			request.Header.Set("Range", "bytes=0-10")
		},
		"no-transform": func(request *http.Request, response *http.Response) {
			response.Header.Set("Cache-Control", "public, no-transform")
		},
		"event-stream": func(request *http.Request, response *http.Response) {
			response.Header.Set("Content-Type", "text/event-stream")
		},
		"image": func(request *http.Request, response *http.Response) {
			response.Header.Set("Content-Type", "image/png")
		},
	} {
		t.Run(name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, "http://demo.local", nil)
			request.Header.Set("Accept-Encoding", "gzip, zstd")
			response := &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/plain"},
				},
			}
			configure(request, response)

			if ShouldCompressWebResponse(request, response) {
				t.Fatal("ShouldCompressWebResponse() = true, want false")
			}
		})
	}
}
