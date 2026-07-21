package assets

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"path"
	"sync"

	"github.com/klauspost/compress/zstd"
	"go.yorun.ai/vine/util/vpre"
)

type _EmbedAccessor struct {
	fsys fs.FS
	root string

	mutex  sync.RWMutex
	cached map[string]string
}

func NewEmbedAccessor(fsys fs.FS, root string) Accessor {
	return &_EmbedAccessor{
		fsys:   fsys,
		root:   root,
		cached: map[string]string{},
	}
}

func (a *_EmbedAccessor) Open(filename string, encodings []_Encoding) (*_File, bool) {
	if cachedFilename, exists := a.cachedFilename(filename); exists {
		return a.open(cachedFilename, acceptedFileEncoding(cachedFilename, encodings))
	}

	zstdFilename := filename + ".zst"
	asset, ok := a.open(zstdFilename, acceptedFileEncoding(zstdFilename, encodings))
	if ok {
		a.cacheFilename(filename, zstdFilename)
		return asset, true
	}

	gzipFilename := filename + ".gz"
	asset, ok = a.open(gzipFilename, acceptedFileEncoding(gzipFilename, encodings))
	if ok {
		a.cacheFilename(filename, gzipFilename)
		return asset, true
	}

	asset, ok = a.open(filename, encodingNone)
	if ok {
		a.cacheFilename(filename, filename)
	}
	return asset, ok
}

func (a *_EmbedAccessor) cachedFilename(filename string) (string, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	cached, exists := a.cached[filename]
	return cached, exists
}

func (a *_EmbedAccessor) cacheFilename(original string, target string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.cached[original] = target
}

func (a *_EmbedAccessor) open(filename string, encoding _Encoding) (*_File, bool) {
	filePath := path.Join(a.root, filename)
	info, err := fs.Stat(a.fsys, filePath)
	if err != nil {
		return nil, false
	}

	content, err := fs.ReadFile(a.fsys, filePath)
	vpre.CheckNilError(err, "read embedded static asset %s failed", filePath)
	if encoding == encodingNone {
		switch path.Ext(filePath) {
		case ".zst":
			content = decodeZst(content, filePath)
		case ".gz":
			content = decodeGzip(content, filePath)
		}
	}
	return &_File{
		ModTime:  info.ModTime(),
		Encoding: encoding,
		Content:  content,
	}, true
}

func acceptedFileEncoding(filename string, encodings []_Encoding) _Encoding {
	switch path.Ext(filename) {
	case ".zst":
		if acceptsEncoding(encodings, encodingZstd) {
			return encodingZstd
		}
	case ".gz":
		if acceptsEncoding(encodings, encodingGzip) {
			return encodingGzip
		}
	}
	return encodingNone
}

func decodeZst(content []byte, filename string) []byte {
	reader, err := zstd.NewReader(bytes.NewReader(content))
	vpre.CheckNilError(err, "open embedded static zstd %s failed", filename)
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	vpre.CheckNilError(err, "read embedded static zstd %s failed", filename)
	return decoded
}

func decodeGzip(content []byte, filename string) []byte {
	reader, err := gzip.NewReader(bytes.NewReader(content))
	vpre.CheckNilError(err, "open embedded static gzip %s failed", filename)
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	vpre.CheckNilError(err, "read embedded static gzip %s failed", filename)
	return decoded
}
