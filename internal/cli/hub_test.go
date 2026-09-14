package cli

import (
	"fmt"
	"os"
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
		if flags.WatchListen != "127.0.0.1:9091" {
			t.Fatalf("unexpected watch listen: %q", flags.WatchListen)
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
		if flags.MQNatsEndpoint != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQNatsEndpoint)
		}
		if flags.MQEmbedded {
			t.Fatal("unexpected mq-embedded")
		}
		if flags.MTLS.CAFile != "/tmp/ca.pem" || flags.MTLS.CertFile != "/tmp/hub.pem" || flags.MTLS.KeyFile != "/tmp/hub-key.pem" {
			t.Fatalf("unexpected mTLS files: %#v", flags.MTLS)
		}
	}

	result := run([]string{"hub", "serve", "--control-listen", ":9090", "--admin-listen", ":9092", "--watch-listen", "127.0.0.1:9091", "--mq-nats-endpoint", "nats://127.0.0.1:4222", "--seed-hub-data-file", "/tmp/hub.yaml", "--dashboard-url", "https://hub.example.com:8443/admin", "--db-sqlite-file", "/tmp/hub.sqlite", "--mtls-ca-file", "/tmp/ca.pem", "--mtls-cert-file", "/tmp/hub.pem", "--mtls-key-file", "/tmp/hub-key.pem"})

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
		if flags.MQNatsEndpoint != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQNatsEndpoint)
		}
		if flags.MQEmbedded {
			t.Fatal("unexpected mq-embedded")
		}
	}

	result := run([]string{"hub", "serve", "--control-listen", ":7090", "--mq-nats-endpoint", "nats://127.0.0.1:4222", "--db-postgres-url", "postgres://demo:demo@127.0.0.1:5432/hub"})

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
	if !strings.Contains(result.stdout, "--mq-nats-endpoint") {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "--mq-embedded") {
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
	t.Setenv(EnvHubWatchListen, "127.0.0.1:10091")
	t.Setenv(EnvHubMQNatsEndpoint, "nats://127.0.0.1:4222")
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
		if flags.WatchListen != "127.0.0.1:10091" {
			t.Fatalf("unexpected watch listen: %q", flags.WatchListen)
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
		if flags.MQNatsEndpoint != "nats://127.0.0.1:4222" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQNatsEndpoint)
		}
		if flags.MQEmbedded {
			t.Fatal("unexpected mq-embedded")
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
		if !flags.MQEmbedded {
			t.Fatal("expected mq-embedded")
		}
		if flags.MQNatsEndpoint != "" {
			t.Fatalf("unexpected mq endpoint: %q", flags.MQNatsEndpoint)
		}
	}

	result := run([]string{"hub", "serve", "--mq-embedded", "--db-sqlite-file", "/tmp/hub.sqlite"})

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
		{"hub", "serve", "--no-db", "--seed-hub-data-file", "seed.yaml", "--mq-embedded"},
		{"hub", "serve", "--seed-hub-data-file", "seed.yaml", "--mq-embedded"},
	} {
		result := run(args)
		if result.exitCode != exitCodeSuccess {
			t.Fatalf("%s", result.stderr)
		}
	}
}

func TestHubSeedHubInputs(t *testing.T) {
	original := startHubApp
	t.Cleanup(func() { startHubApp = original })
	for _, fromEnv := range []bool{false, true} {
		t.Run(fmt.Sprint(fromEnv), func(t *testing.T) {
			for _, name := range []string{EnvSeedHubDataFile, EnvSeedHubSourceFile, EnvSeedHubVarsFile} {
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

func TestHubWatchListenCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{name: "default", want: "127.0.0.1:7072"},
		{name: "watch flag", args: []string{"--watch-listen", ":8100"}, want: ":8100"},
		{name: "watch environment", env: map[string]string{EnvHubWatchListen: ":8101"}, want: ":8101"},
		{name: "redis flag", args: []string{"--redis-listen", ":8200"}, want: ":8200"},
		{name: "redis environment", env: map[string]string{EnvHubRedisListen: ":8201"}, want: ":8201"},
		{name: "both flags", args: []string{"--watch-listen", ":8100", "--redis-listen", ":8200"}, want: ":8100"},
		{name: "both flags reversed", args: []string{"--redis-listen", ":8200", "--watch-listen", ":8100"}, want: ":8100"},
		{name: "both environments", env: map[string]string{EnvHubWatchListen: ":8101", EnvHubRedisListen: ":8201"}, want: ":8101"},
		{name: "watch flag and redis environment", args: []string{"--watch-listen", ":8100"}, env: map[string]string{EnvHubRedisListen: ":8201"}, want: ":8100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{EnvHubWatchListen, EnvHubRedisListen} {
				t.Setenv(name, "")
				if err := os.Unsetenv(name); err != nil {
					t.Fatal(err)
				}
			}
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			original := startHubApp
			t.Cleanup(func() { startHubApp = original })
			called := false
			startHubApp = func(flags hubconf.Flag) {
				called = true
				if flags.WatchListen != tc.want {
					t.Fatalf("got %q, want %q", flags.WatchListen, tc.want)
				}
			}
			result := run(append([]string{"hub", "serve"}, tc.args...))
			if result.exitCode != exitCodeSuccess || !called {
				t.Fatalf("command failed: %#v", result)
			}
			if result.stderr != "" {
				t.Fatalf("unexpected command error: %q", result.stderr)
			}
		})
	}
}

func TestHubServeHelpMarksRedisListenDeprecated(t *testing.T) {
	result := run([]string{"hub", "serve", "--help"})
	if result.exitCode != exitCodeSuccess || !strings.Contains(result.stdout, "--watch-listen") || !strings.Contains(result.stdout, "deprecated: use --watch-listen") {
		t.Fatalf("unexpected help: %#v", result)
	}
}

func TestHubMQInputsSupportOldNamesAndPreferNewNames(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		env      map[string]string
		embedded bool
		endpoint string
	}{
		{name: "new flags", args: []string{"--mq-embedded"}, embedded: true},
		{name: "old flags", args: []string{"--mq-embedded-nats"}, embedded: true},
		{name: "new false overrides old true", args: []string{"--mq-embedded-nats", "--mq-embedded=false"}},
		{name: "old endpoint", args: []string{"--mq-external-nats-url", "nats://old:4222"}, endpoint: "nats://old:4222"},
		{name: "new endpoint wins", args: []string{"--mq-nats-endpoint", "nats://new:4222", "--mq-external-nats-url", "nats://old:4222"}, endpoint: "nats://new:4222"},
		{name: "old environment", env: map[string]string{EnvHubMQEmbeddedNats: "true"}, embedded: true},
		{name: "new environment", env: map[string]string{EnvHubMQEmbedded: "true"}, embedded: true},
		{name: "new environment false wins", env: map[string]string{EnvHubMQEmbedded: "false", EnvHubMQEmbeddedNats: "true"}},
		{name: "old endpoint environment", env: map[string]string{EnvHubMQExternalNatsURL: "nats://old:4222"}, endpoint: "nats://old:4222"},
		{name: "new endpoint environment wins", env: map[string]string{EnvHubMQExternalNatsURL: "nats://old:4222", EnvHubMQNatsEndpoint: "nats://new:4222"}, endpoint: "nats://new:4222"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{EnvHubMQEmbedded, EnvHubMQEmbeddedNats, EnvHubMQNatsEndpoint, EnvHubMQExternalNatsURL} {
				t.Setenv(key, "")
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			original := startHubApp
			t.Cleanup(func() { startHubApp = original })
			called := false
			startHubApp = func(flags hubconf.Flag) {
				called = true
				if flags.MQEmbedded != tc.embedded || flags.MQNatsEndpoint != tc.endpoint {
					t.Fatalf("got embedded=%v endpoint=%q", flags.MQEmbedded, flags.MQNatsEndpoint)
				}
			}
			result := run(append([]string{"hub", "serve"}, tc.args...))
			if result.exitCode != 0 || !called {
				t.Fatalf("command failed: %+v", result)
			}
		})
	}
}
