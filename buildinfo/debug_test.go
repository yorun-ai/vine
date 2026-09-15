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

func TestMustVineVersion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		info   debug.BuildInfo
		linker string
		want   string
	}{
		{name: "Vine binary", info: debug.BuildInfo{Main: debug.Module{Path: "go.yorun.ai/vine", Version: "v0.17.0"}}, want: "v0.17.0"},
		{name: "release linker version", info: debug.BuildInfo{Main: debug.Module{Path: "go.yorun.ai/vine", Version: "(devel)"}}, linker: "v0.18.0", want: "v0.18.0"},
		{name: "embedded Hub", info: debug.BuildInfo{Main: debug.Module{Path: "example.com/app", Version: "v9.0.0"}, Deps: []*debug.Module{{Path: "go.yorun.ai/vine", Version: "v0.17.0"}}}, linker: "v9.0.0", want: "v0.17.0"},
		{name: "development dependency", info: debug.BuildInfo{Main: debug.Module{Path: "example.com/app", Version: "v9.0.0"}, Deps: []*debug.Module{{Path: "go.yorun.ai/vine", Version: "(devel)"}}}, want: DevVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalRead, originalLinker := readBuildInfo, ldModuleVersion
			t.Cleanup(func() { readBuildInfo, ldModuleVersion = originalRead, originalLinker })
			readBuildInfo = func() (*debug.BuildInfo, bool) { return &tc.info, true }
			ldModuleVersion = tc.linker
			if got := MustVineVersion(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
