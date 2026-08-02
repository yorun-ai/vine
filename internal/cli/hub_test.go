package cli

import (
	"strings"
	"testing"

	hubconf "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func TestRunHubServe(t *testing.T) {
	originalStart := startHubApp
	defer func() { startHubApp = originalStart }()

	called := false
	startHubApp = func(flags hubconf.Flag) {
		called = true
		if flags.ControlListen != ":9090" {
			t.Fatalf("unexpected control listen: %q", flags.ControlListen)
		}
		if flags.ManagementListen != ":9092" {
			t.Fatalf("unexpected management listen: %q", flags.ManagementListen)
		}
		if flags.RedisListen != "127.0.0.1:9091" {
			t.Fatalf("unexpected redis listen: %q", flags.RedisListen)
		}
		if flags.DBSQLiteFile != "/tmp/hub.sqlite" {
			t.Fatalf("unexpected sqlitePath: %q", flags.DBSQLiteFile)
		}
		if flags.SeedYAMLPath != "/tmp/hub.yaml" {
			t.Fatalf("unexpected seed yaml path: %q", flags.SeedYAMLPath)
		}
		if flags.DashboardURLRaw != "https://hub.example.com:8443/admin" {
			t.Fatalf("unexpected dashboard url raw: %q", flags.DashboardURLRaw)
		}
		if flags.DashboardURLSet {
			t.Fatal("unexpected dashboard url set before normalize")
		}
		if flags.DashboardURL != nil {
			t.Fatalf("unexpected dashboard url before normalize: %q", flags.DashboardURL)
		}
		if flags.MQExternalNatsURL != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQExternalNatsURL)
		}
		if flags.MQEmbeddedNats {
			t.Fatal("unexpected mq-embedded-nats")
		}
	}

	result := run([]string{"hub", "serve", "--control-listen", ":9090", "--management-listen", ":9092", "--redis-listen", "127.0.0.1:9091", "--mq-external-nats-url", "nats://127.0.0.1:4222", "--seed-yaml-file", "/tmp/hub.yaml", "--dashboard-url", "https://hub.example.com:8443/admin", "--db-sqlite-file", "/tmp/hub.sqlite"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
	if !called {
		t.Fatal("expected hub app to start")
	}
}

func TestRunHubServeRejectsRemovedAPIListenFlag(t *testing.T) {
	originalStart := startHubApp
	defer func() { startHubApp = originalStart }()

	called := false
	startHubApp = func(hubconf.Flag) {
		called = true
	}

	result := run([]string{"hub", "serve", "--api-listen", ":9090"})

	if result.exitCode != exitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if called {
		t.Fatal("hub app started with removed --api-listen flag")
	}
}

func TestRunHubServePG(t *testing.T) {
	originalStart := startHubApp
	defer func() { startHubApp = originalStart }()

	called := false
	startHubApp = func(flags hubconf.Flag) {
		called = true
		if flags.ControlListen != ":7090" {
			t.Fatalf("unexpected control listen: %q", flags.ControlListen)
		}
		if flags.ManagementListen != hubconf.HubDefaultManagementListen {
			t.Fatalf("unexpected management listen: %q", flags.ManagementListen)
		}
		if flags.DBPostgresURL != "postgres://demo:demo@127.0.0.1:5432/hub" {
			t.Fatalf("unexpected pgConnUrl: %q", flags.DBPostgresURL)
		}
		if flags.MQExternalNatsURL != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQExternalNatsURL)
		}
		if flags.MQEmbeddedNats {
			t.Fatal("unexpected mq-embedded-nats")
		}
	}

	result := run([]string{"hub", "serve", "--control-listen", ":7090", "--mq-external-nats-url", "nats://127.0.0.1:4222", "--db-postgres-url", "postgres://demo:demo@127.0.0.1:5432/hub"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if !called {
		t.Fatal("expected hub app to start")
	}
}

func TestRunHubHelpShowsServeOptions(t *testing.T) {
	result := run([]string{"hub", "--help"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stdout, "serve OPTIONS:") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if strings.Contains(result.stdout, "--api-listen") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--control-listen") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--management-listen") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--mq-external-nats-url") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--mq-embedded-nats") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--seed-yaml-file") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--dashboard-url") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--db-sqlite-file") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--db-postgres-url") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
}

func TestRunHubServeFromEnv(t *testing.T) {
	originalStart := startHubApp
	defer func() { startHubApp = originalStart }()

	t.Setenv(EnvHubControlListen, ":10090")
	t.Setenv(EnvHubManagementListen, ":10092")
	t.Setenv(EnvHubRedisListen, "127.0.0.1:10091")
	t.Setenv(EnvHubMQExternalNatsURL, "nats://127.0.0.1:4222")
	t.Setenv(EnvHubSeedYAMLFile, "/tmp/env-hub.yaml")
	t.Setenv(EnvHubDashboardURL, "http://:10099")
	t.Setenv(EnvHubDBSQLiteFile, "/tmp/env-hub.sqlite")

	called := false
	startHubApp = func(flags hubconf.Flag) {
		called = true
		if flags.ControlListen != ":10090" {
			t.Fatalf("unexpected control listen: %q", flags.ControlListen)
		}
		if flags.ManagementListen != ":10092" {
			t.Fatalf("unexpected management listen: %q", flags.ManagementListen)
		}
		if flags.RedisListen != "127.0.0.1:10091" {
			t.Fatalf("unexpected redis listen: %q", flags.RedisListen)
		}
		if flags.DBSQLiteFile != "/tmp/env-hub.sqlite" {
			t.Fatalf("unexpected sqlitePath: %q", flags.DBSQLiteFile)
		}
		if flags.SeedYAMLPath != "/tmp/env-hub.yaml" {
			t.Fatalf("unexpected seed yaml path: %q", flags.SeedYAMLPath)
		}
		if flags.DashboardURLRaw != "http://:10099" {
			t.Fatalf("unexpected dashboard url raw: %q", flags.DashboardURLRaw)
		}
		if flags.DashboardURLSet {
			t.Fatal("unexpected dashboard url set before normalize")
		}
		if flags.DashboardURL != nil {
			t.Fatalf("unexpected dashboard url before normalize: %q", flags.DashboardURL)
		}
		if flags.MQExternalNatsURL != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQExternalNatsURL)
		}
		if flags.MQEmbeddedNats {
			t.Fatal("unexpected mq-embedded-nats")
		}
	}

	result := run([]string{"hub", "serve"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if !called {
		t.Fatal("expected hub app to start")
	}
}

func TestRunHubServeEnableNats(t *testing.T) {
	originalStart := startHubApp
	defer func() { startHubApp = originalStart }()

	called := false
	startHubApp = func(flags hubconf.Flag) {
		called = true
		if !flags.MQEmbeddedNats {
			t.Fatal("expected mq-embedded-nats")
		}
		if flags.MQExternalNatsURL != "" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQExternalNatsURL)
		}
	}

	result := run([]string{"hub", "serve", "--mq-embedded-nats", "--db-sqlite-file", "/tmp/hub.sqlite"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if !called {
		t.Fatal("expected hub app to start")
	}
}
