package app

import (
	"log"
	"os"
	"path/filepath"

	"strings"
	"testing"

	"go.yorun.ai/vine/buildinfo"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"

	"go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

func TestPortalAppFiltersOnlyTLSHandshakeDisconnects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog.jsonl")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelInfo,
		OutputPath: path,
	}))
	t.Cleanup(func() {
		logger.SetDefault(logger.New("vine:default"))
	})

	dropped := []string{
		"http: TLS handshake error from 127.0.0.1:1234: EOF",
		"http: TLS handshake error from 10.202.70.24:36500: read tcp 172.22.101.51:443->10.202.70.24:36500: read: connection reset by peer",
		"http: TLS handshake error from [::1]:1234: read tcp [::1]:443->[::1]:1234: read: connection reset by peer",
	}
	retained := []string{
		"http: TLS handshake error from 127.0.0.1:1234: remote error: tls: bad certificate",
		"http: TLS handshake error from 127.0.0.1:1234: tls: first record does not look like a TLS handshake",
		"http: TLS handshake error from 127.0.0.1:1234: i/o timeout",
		"http: TLS handshake error from 127.0.0.1:1234: EOF: extra detail",
		"http: TLS handshake error from 127.0.0.1:1234: read tcp 127.0.0.1:443->127.0.0.1:1234: read: connection reset by peer: extra detail",
		"read tcp 127.0.0.1:443->127.0.0.1:1234: read: connection reset by peer",
		"unrelated standard library INFO",
	}
	for _, message := range dropped {
		log.Print(message)
	}
	for _, message := range retained {
		log.Print(message)
	}

	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range dropped {
		if strings.Contains(string(output), `"msg":"`+message+`"`) {
			t.Errorf("handshake disconnect was logged: %q", message)
		}
	}
	for _, message := range retained {
		if !strings.Contains(string(output), message) {
			t.Errorf("standard log message missing: %q", message)
		}
	}
}

func TestPortalAppKeepsTLSHandshakeDisconnectsAtDebug(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdlog.jsonl")
	logger.SetDefault(logger.New("vine:test", logger.WithOption{
		Format:     logger.FormatJSON,
		Level:      logger.LevelDebug,
		OutputPath: path,
	}))
	t.Cleanup(func() {
		logger.SetDefault(logger.New("vine:default"))
	})
	message := "http: TLS handshake error from 10.202.70.24:36500: read tcp 172.22.101.51:443->10.202.70.24:36500: read: connection reset by peer"
	log.Print(message)
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `"level":"DEBUG"`) || strings.Contains(string(output), `"level":"INFO"`) || strings.Count(string(output), `"msg":"`+message+`"`) != 1 {
		t.Fatalf("handshake disconnect should appear once at Debug only: %s", output)
	}
}

func TestPortalAppDIInitSetsRunFlagListenAddr(t *testing.T) {
	spec := &PortalApp{
		AppFlag:    &internalapp.RunFlag{},
		InprocFlag: &internalapp.InternalInprocFlag{},
		Flag:       &flag.Flag{HubEndpoint: "http://demo.local:7071"},
	}

	spec.DIInit()

	if got, want := spec.InternalAttrs.CurrentApp.Version(), buildinfo.MustVineVersion(); got != want {
		t.Fatalf("unexpected daemon version: got %q, want Vine version %q", got, want)
	}

	if got := spec.AppFlag.ListenAddr; got != "" {
		t.Fatalf("expected empty listen addr, got %q", got)
	}
	if spec.InternalAttrs.Linker == nil {
		t.Fatal("expected noop linker")
	}
	if got := spec.InternalAttrs.Linker.RpcProxyEndpoint(); got != "http://demo.local:7071/rpc/invoke" {
		t.Fatalf("unexpected hub redirect endpoint: %s", got)
	}
	if !spec.InternalAttrs.DisableConsole {
		t.Fatal("expected console to be disabled")
	}
	if !spec.InternalAttrs.DisableHTTPServer {
		t.Fatal("expected http server to be disabled")
	}
	if spec.InternalAttrs.InprocHostPath != "" {
		t.Fatalf("expected empty inproc host path, got %q", spec.InternalAttrs.InprocHostPath)
	}
}
