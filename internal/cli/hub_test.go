package cli

import (
	"fmt"
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
		if flags.AdminListen != ":9092" {
			t.Fatalf("unexpected admin listen: %q", flags.AdminListen)
		}
		if flags.RedisListen != "127.0.0.1:9091" {
			t.Fatalf("unexpected redis listen: %q", flags.RedisListen)
		}
		if flags.DBSQLiteFile != "/tmp/hub.sqlite" {
			t.Fatalf("unexpected sqlitePath: %q", flags.DBSQLiteFile)
		}
		if flags.SeedHubDataFile != "/tmp/hub.yaml" {
			t.Fatalf("unexpected seed yaml path: %q", flags.SeedHubDataFile)
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
		if flags.MTLS.CAFile != "/tmp/ca.pem" || flags.MTLS.CertFile != "/tmp/hub.pem" || flags.MTLS.KeyFile != "/tmp/hub-key.pem" {
			t.Fatalf("unexpected mTLS files: %#v", flags.MTLS)
		}
	}

	result := run([]string{"hub", "serve", "--control-listen", ":9090", "--admin-listen", ":9092", "--redis-listen", "127.0.0.1:9091", "--mq-external-nats-url", "nats://127.0.0.1:4222", "--seed-hub-data-file", "/tmp/hub.yaml", "--dashboard-url", "https://hub.example.com:8443/admin", "--db-sqlite-file", "/tmp/hub.sqlite", "--mtls-ca-file", "/tmp/ca.pem", "--mtls-cert-file", "/tmp/hub.pem", "--mtls-key-file", "/tmp/hub-key.pem"})

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
		if flags.AdminListen != hubconf.HubDefaultAdminListen {
			t.Fatalf("unexpected admin listen: %q", flags.AdminListen)
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
	if !strings.Contains(result.stdout, "--admin-listen") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--mq-external-nats-url") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--mq-embedded-nats") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--seed-hub-data-file") {
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
	t.Setenv(EnvHubAdminListen, ":10092")
	t.Setenv(EnvHubRedisListen, "127.0.0.1:10091")
	t.Setenv(EnvHubMQExternalNatsURL, "nats://127.0.0.1:4222")
	t.Setenv(EnvSeedHubDataFile, "/tmp/env-hub.yaml")
	t.Setenv(EnvHubDashboardURL, "http://:10099")
	t.Setenv(EnvHubDBSQLiteFile, "/tmp/env-hub.sqlite")

	called := false
	startHubApp = func(flags hubconf.Flag) {
		called = true
		if flags.ControlListen != ":10090" {
			t.Fatalf("unexpected control listen: %q", flags.ControlListen)
		}
		if flags.AdminListen != ":10092" {
			t.Fatalf("unexpected admin listen: %q", flags.AdminListen)
		}
		if flags.RedisListen != "127.0.0.1:10091" {
			t.Fatalf("unexpected redis listen: %q", flags.RedisListen)
		}
		if flags.DBSQLiteFile != "/tmp/env-hub.sqlite" {
			t.Fatalf("unexpected sqlitePath: %q", flags.DBSQLiteFile)
		}
		if flags.SeedHubDataFile != "/tmp/env-hub.yaml" {
			t.Fatalf("unexpected seed yaml path: %q", flags.SeedHubDataFile)
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

func TestRunHubServeNoDB(t *testing.T) {
	original := startHubApp
	t.Cleanup(func() { startHubApp = original })
	startHubApp = func(flags hubconf.Flag) {
		flags.Normalize(false)
		if !flags.NoDB || flags.Store != hubconf.StoreMemory {
			t.Fatal("expected no-db")
		}
	}
	for _, args := range [][]string{
		{"hub", "serve", "--no-db", "--seed-hub-data-file", "seed.yaml", "--mq-embedded-nats"},
		{"hub", "serve", "--seed-hub-data-file", "seed.yaml", "--mq-embedded-nats"},
	} {
		result := run(args)
		if result.exitCode != exitCodeSuccess {
			t.Fatalf("%s", result.stderr)
		}
	}
}

func TestHubDeprecatedSeedDataFileInputs(t *testing.T) {
	original := startHubApp
	t.Cleanup(func() { startHubApp = original })
	for _, legacyEnv := range []bool{false, true} {
		t.Run(fmt.Sprint(legacyEnv), func(t *testing.T) {
			t.Setenv(EnvSeedHubDataFile, "")
			t.Setenv(EnvHubSeedYAMLFile, "")
			called := false
			startHubApp = func(flags hubconf.Flag) {
				flags.Normalize(true)
				called = true
				if flags.SeedHubDataFile != "old.yaml" {
					t.Fatalf("incorrect path: %s", flags.SeedHubDataFile)
				}
			}
			args := []string{"hub", "serve"}
			if legacyEnv {
				t.Setenv(EnvHubSeedYAMLFile, "old.yaml")
			} else {
				args = append(args, "--seed-yaml-file", "old.yaml")
			}
			result := run(args)
			if result.exitCode != exitCodeSuccess || !called {
				t.Fatalf("legacy input failed: %#v", result)
			}
		})
	}
}

func TestHubSeedHubInputs(t *testing.T) {
	original := startHubApp
	t.Cleanup(func() { startHubApp = original })
	for _, fromEnv := range []bool{false, true} {
		t.Run(fmt.Sprint(fromEnv), func(t *testing.T) {
			for _, name := range []string{EnvSeedHubDataFile, EnvSeedHubSourceFile, EnvSeedHubVarsFile, EnvHubSeedYAMLFile} {
				t.Setenv(name, "")
			}
			called := false
			startHubApp = func(flags hubconf.Flag) {
				called = true
				if flags.SeedHubDataFile != "data.yaml" || flags.SeedHubSourceFile != "source.yaml" || flags.SeedHubVarsFile != "vars.yaml" {
					t.Fatalf("incorrect input mapping: %#v", flags)
				}
			}
			args := []string{"hub", "serve"}
			if fromEnv {
				t.Setenv(EnvSeedHubDataFile, "data.yaml")
				t.Setenv(EnvSeedHubSourceFile, "source.yaml")
				t.Setenv(EnvSeedHubVarsFile, "vars.yaml")
			} else {
				args = append(args, "--seed-hub-data-file", "data.yaml", "--seed-hub-source-file", "source.yaml", "--seed-hub-vars-file", "vars.yaml")
			}
			result := run(args)
			if result.exitCode != exitCodeSuccess || !called {
				t.Fatalf("new seed flags failed: %#v", result)
			}
		})
	}
}
