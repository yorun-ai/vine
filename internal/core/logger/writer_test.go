package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCreatesOutputPathParentDirectories(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "nested", "log", "vine.log")
	log := New("vine:test", WithOption{
		Format:     FormatText,
		Level:      LevelInfo,
		OutputPath: outputPath,
	})
	log.Info("nested output")

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "nested output") {
		t.Fatalf("output file does not contain the logged message: %q", content)
	}
}

func TestLoggersShareWriterByOutputPath(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "logs", "vine.log")
	first := New("vine:test:first", WithOption{
		Format:     FormatText,
		Level:      LevelInfo,
		OutputPath: outputPath,
	})
	second := New("vine:test:second", WithOption{
		Format:     FormatJSON,
		Level:      LevelInfo,
		OutputPath: filepath.Join(filepath.Dir(outputPath), ".", filepath.Base(outputPath)),
	})

	if first.writer != second.writer {
		t.Fatal("loggers with the same output path should share one writer")
	}
}

func TestGlobalAndExplicitOutputPathShareWriter(t *testing.T) {
	resetGlobalOptionForTest(t)
	outputPath := filepath.Join(t.TempDir(), "vine.log")
	SetGlobalOutputPath(outputPath)

	explicit := New("vine:test", WithOption{
		Format:     FormatText,
		Level:      LevelInfo,
		OutputPath: outputPath,
	})
	globalWriter.mutex.RLock()
	globalPathWriter := globalWriter.writer
	globalWriter.mutex.RUnlock()
	if explicit.writer != globalPathWriter {
		t.Fatal("global and explicit output paths should share one writer")
	}
}

func TestOutputPathFailurePanics(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertPanics(t, func() {
		New("vine:test", WithOption{
			OutputPath: filepath.Join(parentFile, "vine.log"),
		})
	})
}

func TestLoggerFollowsGlobalFormatAndOutputPathChanges(t *testing.T) {
	resetGlobalOptionForTest(t)
	SetGlobalLevel(LevelInfo)
	SetGlobalFormat(FormatText)
	firstPath := filepath.Join(t.TempDir(), "first", "vine.log")
	SetGlobalOutputPath(firstPath)

	log := New("vine:test")
	if log.writer != globalWriter {
		t.Fatal("logger without an output path should use the global writer")
	}
	log.Info("before-global-change")

	secondPath := filepath.Join(t.TempDir(), "second", "vine.log")
	SetGlobalFormat(FormatJSON)
	SetGlobalOutputPath(secondPath)
	log.Child("child").Info("after-global-change")

	firstContent, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(firstContent), "msg=before-global-change") ||
		!strings.Contains(string(firstContent), "logger=vine:test") {
		t.Fatalf("first output is not text: %q", firstContent)
	}

	secondContent, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(secondContent), `"msg":"after-global-change"`) ||
		!strings.Contains(string(secondContent), `"logger":"vine:test:child"`) {
		t.Fatalf("second output is not JSON: %q", secondContent)
	}
}

func TestExplicitFormatAndOutputPathDoNotFollowGlobalChanges(t *testing.T) {
	resetGlobalOptionForTest(t)
	SetGlobalFormat(FormatText)
	SetGlobalOutputPath(filepath.Join(t.TempDir(), "global.log"))

	explicitPath := filepath.Join(t.TempDir(), "explicit", "vine.log")
	log := New("vine:test", WithOption{
		Format:     FormatText,
		Level:      LevelInfo,
		OutputPath: explicitPath,
	})

	SetGlobalFormat(FormatJSON)
	SetGlobalOutputPath(filepath.Join(t.TempDir(), "changed-global.log"))
	log.Info("fixed-output")

	content, err := os.ReadFile(explicitPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "msg=fixed-output") {
		t.Fatalf("explicit output did not remain text: %q", content)
	}
}
