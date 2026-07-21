package assets

import (
	"bytes"
	"compress/gzip"
	"testing"
	"testing/fstest"
	"time"

	"github.com/klauspost/compress/zstd"
)

func TestNewEmbedAccessorOpen(t *testing.T) {
	modTime := time.Unix(1700000000, 0)
	accessor := NewEmbedAccessor(fstest.MapFS{
		"dist/index.html.zst": &fstest.MapFile{
			Data:    testZst(t, `<div id="app"></div>`),
			Mode:    0o644,
			ModTime: modTime,
		},
		"dist/assets/style.css.gz": &fstest.MapFile{
			Data:    testGzip(t, `body { color: green; }`),
			Mode:    0o644,
			ModTime: modTime,
		},
		"dist/assets/app.js": &fstest.MapFile{
			Data:    []byte(`console.log("vine")`),
			Mode:    0o644,
			ModTime: modTime,
		},
	}, "dist")
	embedAccessor := accessor.(*_EmbedAccessor)

	asset, ok := accessor.Open(indexPath, nil)
	if !ok {
		t.Fatal("Open() zst failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `<div id="app"></div>` {
		t.Fatalf("unexpected zst asset: %#v", asset)
	}
	if embedAccessor.cached[indexPath] != indexPath+".zst" {
		t.Fatalf("unexpected mapped zst filename: %s", embedAccessor.cached[indexPath])
	}

	asset, ok = accessor.Open(indexPath, []_Encoding{encodingZstd})
	if !ok {
		t.Fatal("Open() encoded zst failed")
	}
	if asset.Encoding != encodingZstd || string(asset.Content) == `<div id="app"></div>` {
		t.Fatalf("unexpected encoded zst asset: %#v", asset)
	}

	asset, ok = accessor.Open("/assets/style.css", nil)
	if !ok {
		t.Fatal("Open() gzip failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `body { color: green; }` {
		t.Fatalf("unexpected gzip asset: %#v", asset)
	}
	if embedAccessor.cached["/assets/style.css"] != "/assets/style.css.gz" {
		t.Fatalf("unexpected mapped gzip filename: %s", embedAccessor.cached["/assets/style.css"])
	}

	asset, ok = accessor.Open("/assets/style.css", []_Encoding{encodingGzip})
	if !ok {
		t.Fatal("Open() encoded gzip failed")
	}
	if asset.Encoding != encodingGzip || string(asset.Content) == `body { color: green; }` {
		t.Fatalf("unexpected encoded gzip asset: %#v", asset)
	}

	asset, ok = accessor.Open("/assets/app.js", nil)
	if !ok {
		t.Fatal("Open() plain failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `console.log("vine")` {
		t.Fatalf("unexpected plain asset: %#v", asset)
	}
	if embedAccessor.cached["/assets/app.js"] != "/assets/app.js" {
		t.Fatalf("unexpected mapped plain filename: %s", embedAccessor.cached["/assets/app.js"])
	}

	if _, ok = accessor.Open("/missing.js", nil); ok {
		t.Fatal("expected missing asset")
	}
}

func TestNewEmbedAccessorOpenPrefersZst(t *testing.T) {
	accessor := NewEmbedAccessor(fstest.MapFS{
		"dist/index.html.zst": &fstest.MapFile{
			Data: testZst(t, `<div id="zstd"></div>`),
			Mode: 0o644,
		},
		"dist/index.html.gz": &fstest.MapFile{
			Data: testGzip(t, `<div id="gzip"></div>`),
			Mode: 0o644,
		},
	}, "dist")

	asset, ok := accessor.Open(indexPath, []_Encoding{encodingGzip, encodingZstd})
	if !ok {
		t.Fatal("Open() failed")
	}
	if asset.Encoding != encodingZstd {
		t.Fatalf("unexpected encoding: %s", asset.Encoding)
	}
}

func testZst(t *testing.T, content string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer, err := zstd.NewWriter(&buffer)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	if _, err = writer.Write([]byte(content)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buffer.Bytes()
}

func testGzip(t *testing.T, content string) []byte {
	t.Helper()

	return testGzipBytes(t, []byte(content))
}

func testGzipBytes(t *testing.T, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buffer.Bytes()
}
