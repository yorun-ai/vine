package assets

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"path"

	"github.com/klauspost/compress/zstd"
	"go.yorun.ai/vine/util/vpre"
)

// Tar zstd accessor

type _TarZstAccessor struct {
	content []byte
	files   map[string]*_File
}

func NewTarZstAccessor(content []byte) Accessor {
	accessor := &_TarZstAccessor{content: content}
	accessor.init()
	return accessor
}

func (a *_TarZstAccessor) Open(filename string, encodings []_Encoding) (*_File, bool) {
	file, exists := a.files[filename]
	if !exists {
		return nil, false
	}
	return file, true
}

func (a *_TarZstAccessor) init() {
	reader, err := zstd.NewReader(bytes.NewReader(a.content))
	vpre.CheckNilError(err, "open embedded static zstd failed")
	defer reader.Close()

	a.files = readTarAssets(reader)
}

// Tar gzip accessor

type _TarGzipAccessor struct {
	content []byte
	files   map[string]*_File
}

func NewTarGzipAccessor(content []byte) Accessor {
	accessor := &_TarGzipAccessor{content: content}
	accessor.init()
	return accessor
}

func (a *_TarGzipAccessor) Open(filename string, encodings []_Encoding) (*_File, bool) {
	file, exists := a.files[filename]
	if !exists {
		return nil, false
	}
	return file, true
}

func (a *_TarGzipAccessor) init() {
	reader, err := gzip.NewReader(bytes.NewReader(a.content))
	vpre.CheckNilError(err, "open embedded static gzip failed")
	defer reader.Close()

	a.files = readTarAssets(reader)
}

// Zip accessor

type _ZipAccessor struct {
	content []byte
	files   map[string]*zip.File
}

func NewZipAccessor(content []byte) Accessor {
	accessor := &_ZipAccessor{content: content}
	accessor.init()
	return accessor
}

func (a *_ZipAccessor) init() {
	reader, err := zip.NewReader(bytes.NewReader(a.content), int64(len(a.content)))
	vpre.CheckNilError(err, "open embedded static zip failed")

	a.files = map[string]*zip.File{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		a.files[archiveFilename(file.Name)] = file
	}
}

func (a *_ZipAccessor) Open(filename string, encodings []_Encoding) (*_File, bool) {
	file, exists := a.files[filename]
	if !exists {
		return nil, false
	}

	contentReader, err := file.Open()
	vpre.CheckNilError(err, "open embedded static asset %s failed", filename)
	defer contentReader.Close()

	content, err := io.ReadAll(contentReader)
	vpre.CheckNilError(err, "read embedded static asset %s failed", filename)
	return &_File{
		ModTime:  file.Modified,
		Encoding: encodingNone,
		Content:  content,
	}, true
}

// Helpers

func readTarAssets(reader io.Reader) map[string]*_File {
	tarReader := tar.NewReader(reader)
	files := map[string]*_File{}
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		vpre.CheckNilError(err, "read embedded static tar failed")
		if header.FileInfo().IsDir() {
			continue
		}

		name := archiveFilename(header.Name)
		content, err := io.ReadAll(tarReader)
		vpre.CheckNilError(err, "read embedded static asset %s failed", name)
		files[name] = &_File{
			ModTime:  header.ModTime,
			Encoding: encodingNone,
			Content:  content,
		}
	}
	return files
}

// archiveFilename normalizes archive header names once while building the file index.
// Tar commands often store files as ./index.html when archiving the current directory.
func archiveFilename(name string) string {
	return path.Clean("/" + name)
}
