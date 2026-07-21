package buildinfo

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func TestModuleVersion(t *testing.T) {
	for _, tt := range []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "devel", raw: "(devel)", expected: DevVersion},
		{name: "without v prefix", raw: "1.2.3", expected: "v1.2.3"},
		{name: "module version", raw: "v2.3.4", expected: "v2.3.4"},
		{name: "dirty", raw: "v1.1.0-alpha3+dirty", expected: "v1.1.0-alpha3"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := moduleVersion(tt.raw); got != tt.expected {
				t.Fatalf("unexpected module version: got %q want %q", got, tt.expected)
			}
		})
	}
}

func TestIsDevVersion(t *testing.T) {
	if !IsDevVersion(DevVersion) {
		t.Fatalf("expected dev version")
	}
	if IsDevVersion("v1.0.0") {
		t.Fatalf("did not expect release version")
	}
}

func TestMustDebugBuildInfo(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			GoVersion: "go1.26.0",
			Main:      debug.Module{Version: "v1.1.3"},
		}, true
	})

	info := MustDebugBuildInfo()
	if info.Version != "v1.1.3" {
		t.Fatalf("unexpected version: %q", info.Version)
	}
	if info.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("unexpected platform: %q", info.Platform)
	}
	if info.GoVersion != "go1.26.0" {
		t.Fatalf("unexpected go version: %q", info.GoVersion)
	}
}

func TestMustDebugBuildInfoWithDevelVersion(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	})

	info := MustDebugBuildInfo()
	if info.Version != DevVersion {
		t.Fatalf("unexpected version: %q", info.Version)
	}
	if info.GoVersion != "" {
		t.Fatalf("unexpected go version: %q", info.GoVersion)
	}
}

func TestMustDebugBuildInfoRejectsMissingBuildInfo(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return nil, false
	})

	defer assertPanicContains(t, "read Go build info failed")()
	MustDebugBuildInfo()
}

func TestMustVineDependencyVersion(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "go.yorun.ai/vine", Version: "v1.1.5"},
			},
		}, true
	})

	if got := MustVineDependencyVersion(); got != "v1.1.5" {
		t.Fatalf("unexpected dependency version: %q", got)
	}
}

func TestMustVineDependencyVersionUsesDevVersionWhenDependencyIsWorkspaceMainModule(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{}, true
	})

	if got := MustVineDependencyVersion(); got != DevVersion {
		t.Fatalf("unexpected dependency version: %q", got)
	}
}

func TestMustVineDependencyVersionRejectsMissingBuildInfo(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return nil, false
	})

	defer assertPanicContains(t, "read Go build info failed")()
	MustVineDependencyVersion()
}

func TestMustVineDependencyVersionUsesDevVersionForDevelDependency(t *testing.T) {
	setReadBuildInfoForTest(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Deps: []*debug.Module{
				{Path: "go.yorun.ai/vine", Version: "(devel)"},
			},
		}, true
	})

	if got := MustVineDependencyVersion(); got != DevVersion {
		t.Fatalf("unexpected dependency version: %q", got)
	}
}

func setReadBuildInfoForTest(t *testing.T, fn func() (*debug.BuildInfo, bool)) {
	t.Helper()
	original := readBuildInfo
	t.Cleanup(func() {
		readBuildInfo = original
	})
	readBuildInfo = fn
}

func assertPanicContains(t *testing.T, expected string) func() {
	t.Helper()
	return func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("unexpected panic value: %#v", recovered)
		}
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("unexpected panic: got %q want containing %q", err.Error(), expected)
		}
	}
}
