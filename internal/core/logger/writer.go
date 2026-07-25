package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

// Global writer

var globalWriter = &_GlobalWriter{
	writer: os.Stderr,
}

type _GlobalWriter struct {
	mutex      sync.RWMutex
	writer     io.Writer
	outputPath string
}

func (w *_GlobalWriter) Write(p []byte) (int, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.writer.Write(p)
}

func (w *_GlobalWriter) OutputPath() string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.outputPath
}

func (w *_GlobalWriter) SetOutputPath(outputPath string) {
	if w.OutputPath() == outputPath {
		return
	}
	next := sharedLogWriter(outputPath)

	w.mutex.Lock()
	if w.outputPath == outputPath {
		w.mutex.Unlock()
		return
	}
	w.writer = next
	w.outputPath = outputPath
	w.mutex.Unlock()
}

func SetGlobalOutputPath(outputPath string) {
	globalWriter.SetOutputPath(outputPath)
}

// Locked writer

type _LockedWriter struct {
	mutex  sync.Mutex
	writer io.Writer
}

func (w *_LockedWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.writer.Write(p)
}

// Opened writers

type _OpenedWriters struct {
	mutex  sync.Mutex
	byPath map[string]io.Writer
}

var openedWriters = &_OpenedWriters{
	byPath: make(map[string]io.Writer),
}

// sharedLogWriter returns the writer shared by all logger instances that use
// outputPath. Opened files intentionally remain open for the process lifetime:
// loggers have no close lifecycle, and closing a cached writer when the global
// output changes could invalidate other loggers that still use the same path.
func sharedLogWriter(outputPath string) io.Writer {
	if outputPath == "" {
		return os.Stderr
	}

	absolutePath, err := filepath.Abs(outputPath)
	vpre.Check(err == nil, "resolve logger output path %q: %v", outputPath, err)

	openedWriters.mutex.Lock()
	defer openedWriters.mutex.Unlock()
	if writer, ok := openedWriters.byPath[absolutePath]; ok {
		return writer
	}

	err = os.MkdirAll(filepath.Dir(absolutePath), 0o755)
	vpre.Check(err == nil, "create logger output directory for %q: %v", outputPath, err)
	file, err := os.OpenFile(absolutePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	vpre.Check(err == nil, "open logger output %q: %v", outputPath, err)
	writer := &_LockedWriter{writer: io.MultiWriter(os.Stderr, file)}
	openedWriters.byPath[absolutePath] = writer
	return writer
}
