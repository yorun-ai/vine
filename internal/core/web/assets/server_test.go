package assets

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
)

func TestAssetsServerServeAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewTarZstAccessor(testTarZst(t, map[string]string{
		"index.html":    `<div id="app"></div>`,
		"assets/app.js": `console.log("vine")`,
	})))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/assets/app.js")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/javascript; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if recorder.Body.String() != `console.log("vine")` {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestAssetsServerServeAssetsUsesRoutePath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewTarZstAccessor(testTarZst(t, map[string]string{
		"app.js":     `console.log("vine")`,
		"index.html": `<div id="app"></div>`,
	})))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/assets/../app.js")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/../app.js"}}
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != `console.log("vine")` {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestAssetsServerServesArchiveDotSlashFilenames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewTarZstAccessor(testTarZst(t, map[string]string{
		"./index.html": `<div id="app"></div>`,
	})))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/"}}
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Body.String() != `<div id="app"></div>` {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestAssetsServerFallbackAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewTarZstAccessor(testTarZst(t, map[string]string{
		"index.html": `<div id="app"></div>`,
	})))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/portal/site")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/portal/site"}}
	ginCtx.Request.Header.Set("Accept", "text/html")
	server.SetContext(ginCtx)
	server.Serve()
	if recorder.Code != http.StatusOK || recorder.Body.String() != `<div id="app"></div>` {
		t.Fatalf("unexpected fallback response: code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder, ginCtx = newStaticTestContext(http.MethodGet, "/missing.js")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/missing.js"}}
	ginCtx.Request.Header.Set("Accept", "*/*")
	server.SetContext(ginCtx)
	server.Serve()
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing asset status: %d", recorder.Code)
	}

	recorder, ginCtx = newStaticTestContext(http.MethodPost, "/")
	server.SetContext(ginCtx)
	server.Serve()
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected method status: %d", recorder.Code)
	}
}

func TestAssetsServerFallbackMissingIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewTarZstAccessor(testTarZst(t, map[string]string{
		"assets/app.js": `console.log("vine")`,
	})))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/portal/site")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/portal/site"}}
	ginCtx.Request.Header.Set("Accept", "text/html")
	server.SetContext(ginCtx)
	server.Serve()
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing fallback status: %d", recorder.Code)
	}
}

func TestAssetsServerServesAcceptedEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewEmbedAccessor(testEmbedFS(t, map[string][]byte{
		"dist/index.html.zst": testZst(t, `<div id="app"></div>`),
	}), "dist"))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/portal/site")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/portal/site"}}
	ginCtx.Request.Header.Set("Accept", "text/html")
	ginCtx.Request.Header.Set("Accept-Encoding", "gzip, zstd")
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if encoding := recorder.Header().Get("Content-Encoding"); encoding != string(encodingZstd) {
		t.Fatalf("unexpected content encoding: %s", encoding)
	}
	if recorder.Body.String() == `<div id="app"></div>` {
		t.Fatal("expected encoded response body")
	}
}

func TestAssetsServerEncodesPlainAssetWithPreferredEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewEmbedAccessor(testEmbedFS(t, map[string][]byte{
		"dist/assets/app.js": []byte(`console.log("vine")`),
	}), "dist"))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/assets/app.js")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}
	ginCtx.Request.Header.Set("Accept-Encoding", "gzip, zstd")
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if encoding := recorder.Header().Get("Content-Encoding"); encoding != string(encodingZstd) {
		t.Fatalf("unexpected content encoding: %s", encoding)
	}
	if recorder.Body.String() == `console.log("vine")` {
		t.Fatal("expected zstd response body")
	}
}

func TestAssetsServerEncodesPlainAssetWithGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer(NewEmbedAccessor(testEmbedFS(t, map[string][]byte{
		"dist/assets/app.js": []byte(`console.log("vine")`),
	}), "dist"))

	recorder, ginCtx := newStaticTestContext(http.MethodGet, "/assets/app.js")
	ginCtx.Params = gin.Params{{Key: "path", Value: "/assets/app.js"}}
	ginCtx.Request.Header.Set("Accept-Encoding", "gzip")
	server.SetContext(ginCtx)
	server.Serve()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if encoding := recorder.Header().Get("Content-Encoding"); encoding != string(encodingGzip) {
		t.Fatalf("unexpected content encoding: %s", encoding)
	}
	if recorder.Body.String() == `console.log("vine")` {
		t.Fatal("expected gzip response body")
	}
}

func newStaticTestContext(method string, path string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(method, "http://demo.local"+path, nil)
	return recorder, ginCtx
}

func testEmbedFS(t *testing.T, files map[string][]byte) fstest.MapFS {
	t.Helper()

	fsys := fstest.MapFS{}
	modTime := time.Unix(1700000000, 0)
	for name, content := range files {
		fsys[name] = &fstest.MapFile{
			Data:    content,
			Mode:    0o644,
			ModTime: modTime,
		}
	}
	return fsys
}

func testTarZst(t *testing.T, files map[string]string) []byte {
	t.Helper()

	tarContent := testTar(t, files)
	var zstBuffer bytes.Buffer
	zstWriter, err := zstd.NewWriter(&zstBuffer)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	if _, err = zstWriter.Write(tarContent); err != nil {
		t.Fatalf("zstd Write() error = %v", err)
	}
	if err = zstWriter.Close(); err != nil {
		t.Fatalf("zstd Close() error = %v", err)
	}
	return zstBuffer.Bytes()
}

func testTarGzip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	return testGzipBytes(t, testTar(t, files))
}

func testTar(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var tarBuffer bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuffer)
	modTime := time.Unix(1700000000, 0)
	for name, content := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name:    name,
			Mode:    0o644,
			Size:    int64(len(content)),
			ModTime: modTime,
		})
		if err != nil {
			t.Fatalf("WriteHeader() error = %v", err)
		}
		if _, err = tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return tarBuffer.Bytes()
}

func testZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)
	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if _, err = writer.Write([]byte(content)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return zipBuffer.Bytes()
}
