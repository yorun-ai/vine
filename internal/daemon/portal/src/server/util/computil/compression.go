package computil

import (
	"bytes"
	"compress/gzip"
	"mime"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"

	"go.yorun.ai/vine/internal/util/httputil"
	"go.yorun.ai/vine/util/vpre"
)

const (
	EncodingGzip          = "gzip"
	EncodingZstd          = "zstd"
	CompressionThreshold  = 4 * 1024
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentRange    = "Content-Range"
	headerRange           = "Range"
)

func CompressResponseBody(body []byte, acceptEncoding string) ([]byte, string) {
	if len(body) <= CompressionThreshold {
		return body, ""
	}
	encoding := PreferredEncoding(acceptEncoding)
	switch encoding {
	case EncodingZstd:
		return zstdResponseBody(body), encoding
	case EncodingGzip:
		return gzipResponseBody(body), encoding
	default:
		return body, ""
	}
}

func PreferredEncoding(acceptEncoding string) string {
	accepted := acceptedEncodings(acceptEncoding)
	if accepted[EncodingZstd] {
		return EncodingZstd
	}
	if accepted[EncodingGzip] {
		return EncodingGzip
	}
	return ""
}

func ShouldCompressWebResponse(request *http.Request, response *http.Response) bool {
	if request.Method == http.MethodHead || response.StatusCode != http.StatusOK {
		return false
	}
	if PreferredEncoding(request.Header.Get(headerAcceptEncoding)) == "" {
		return false
	}
	if response.Header.Get(headerContentEncoding) != "" {
		return false
	}
	if request.Header.Get(headerRange) != "" || response.Header.Get(headerContentRange) != "" {
		return false
	}
	if hasNoTransform(response.Header.Get("Cache-Control")) {
		return false
	}
	if httputil.IsEventStreamResponse(response) {
		return false
	}
	return isCompressibleContentType(response.Header.Get("Content-Type"))
}

func acceptedEncodings(acceptEncoding string) map[string]bool {
	accepted := map[string]bool{}
	for _, item := range strings.Split(acceptEncoding, ",") {
		encoding, params, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil || params["q"] == "0" {
			continue
		}
		if encoding == EncodingZstd || encoding == EncodingGzip {
			accepted[encoding] = true
		}
	}
	return accepted
}

func hasNoTransform(cacheControl string) bool {
	for _, item := range strings.Split(cacheControl, ",") {
		if strings.EqualFold(strings.TrimSpace(item), "no-transform") {
			return true
		}
	}
	return false
}

func isCompressibleContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType, _, _ = strings.Cut(contentType, ";")
		mediaType = strings.TrimSpace(mediaType)
	}
	mediaType = strings.ToLower(mediaType)
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	if mediaType == "image/svg+xml" || mediaType == "application/wasm" {
		return true
	}
	if mediaType == "application/json" || mediaType == "application/javascript" || mediaType == "application/xml" {
		return true
	}
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml")
}

func zstdResponseBody(body []byte) []byte {
	var buffer bytes.Buffer
	writer, err := zstd.NewWriter(&buffer)
	vpre.CheckNilError(err, "open response zstd writer failed")
	_, err = writer.Write(body)
	vpre.CheckNilError(err, "write response zstd body failed")
	err = writer.Close()
	vpre.CheckNilError(err, "close response zstd writer failed")
	return buffer.Bytes()
}

func gzipResponseBody(body []byte) []byte {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(body)
	vpre.CheckNilError(err, "write response gzip body failed")
	err = writer.Close()
	vpre.CheckNilError(err, "close response gzip writer failed")
	return buffer.Bytes()
}
