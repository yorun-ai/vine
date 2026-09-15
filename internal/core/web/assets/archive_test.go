package assets

import "testing"

func TestNewTarZstAssetsAccessorOpen(t *testing.T) {
	content := testTarZst(t, map[string]string{
		"index.html":    `<div id="app"></div>`,
		"assets/app.js": `console.log("vine")`,
	})
	store := NewTarZstAccessor(content)
	tarZstStore := store.(*_TarZstAccessor)
	if string(tarZstStore.files["/assets/app.js"].Content) != `console.log("vine")` {
		t.Fatal("expected asset content to load on init")
	}

	asset, ok := store.Open("/assets/app.js", []_Encoding{encodingZstd})
	if !ok {
		t.Fatal("Open() failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `console.log("vine")` {
		t.Fatalf("unexpected asset: %#v", asset)
	}

	if _, ok = store.Open("/missing.js", nil); ok {
		t.Fatal("expected missing asset")
	}
}

func TestNewTarGzipAssetsAccessorOpen(t *testing.T) {
	content := testTarGzip(t, map[string]string{
		"index.html":    `<div id="app"></div>`,
		"assets/app.js": `console.log("vine")`,
	})
	store := NewTarGzipAccessor(content)
	tarGzipStore := store.(*_TarGzipAccessor)
	if string(tarGzipStore.files["/assets/app.js"].Content) != `console.log("vine")` {
		t.Fatal("expected asset content to load on init")
	}

	asset, ok := store.Open("/assets/app.js", []_Encoding{encodingGzip})
	if !ok {
		t.Fatal("Open() failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `console.log("vine")` {
		t.Fatalf("unexpected asset: %#v", asset)
	}

	if _, ok = store.Open("/missing.js", nil); ok {
		t.Fatal("expected missing asset")
	}
}

func TestNewZipAccessorOpen(t *testing.T) {
	content := testZip(t, map[string]string{
		"index.html":    `<div id="app"></div>`,
		"assets/app.js": `console.log("vine")`,
	})
	store := NewZipAccessor(content)
	zipStore := store.(*_ZipAccessor)
	if zipStore.files["/assets/app.js"] == nil {
		t.Fatal("expected zip filename index to load on init")
	}

	asset, ok := store.Open("/assets/app.js", nil)
	if !ok {
		t.Fatal("Open() failed")
	}
	if asset.Encoding != encodingNone || string(asset.Content) != `console.log("vine")` {
		t.Fatalf("unexpected asset: %#v", asset)
	}

	if _, ok = store.Open("/missing.js", nil); ok {
		t.Fatal("expected missing asset")
	}
}
