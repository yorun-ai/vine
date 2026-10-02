package cli

import (
	"context"
	"os"
	stdRuntime "runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/core/logger"
	"go.yorun.ai/vine/core/skel"
	"go.yorun.ai/vine/internal/appcli"
	hubflag "go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
	portalflag "go.yorun.ai/vine/internal/daemon/portal/src/server/flag"
)

func TestRunVersion(t *testing.T) {
	result := run([]string{"version"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	expected := versionInfo().TextString() + "\n"
	if result.stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if want := "  Platform   " + stdRuntime.GOOS + "/" + stdRuntime.GOARCH + "\n"; !strings.Contains(result.stdout, want) {
		t.Fatalf("expected stdout to contain platform line %q, got %q", want, result.stdout)
	}
	if want := "  MinSkelcVersion  " + skel.MinSkelcVersion() + "\n"; !strings.Contains(result.stdout, want) {
		t.Fatalf("expected stdout to contain dependency line %q, got %q", want, result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestRunVersionJSON(t *testing.T) {
	result := run([]string{"version", "--json"})

	if result.exitCode != exitCodeSuccess {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	expected := versionInfo().JSONString() + "\n"
	if result.stdout != expected {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

// Exercise every environment-backed component flag: old names are ignored,
// scoped empty values are preserved, and command-line values take precedence.
func TestComponentEnvironmentPrecedence(t *testing.T) {
	for component, create := range map[string]func() *ucli.Command{
		"HUB": newHubServeCommand, "LINK": newLinkServeCommand, "PORTAL": newPortalServeCommand,
	} {
		for _, declared := range create().Flags {
			var sources ucli.ValueSourceChain
			switch flag := declared.(type) {
			case *ucli.StringFlag:
				sources = flag.Sources
			case *ucli.BoolFlag:
				sources = flag.Sources
			case *ucli.StringSliceFlag:
				sources = flag.Sources
			case *appcli.RepeatedStringFlag:
				sources = flag.Sources
			default:
				t.Fatalf("unexpected flag %T", flag)
			}
			keys := sources.EnvKeys()
			if len(keys) != 1 || !strings.HasPrefix(keys[0], "VINE_"+component+"_") {
				t.Fatalf("unexpected sources for %s: %v", declared.Names()[0], keys)
			}
			for _, scenario := range []string{"old-only", "scoped", "empty", "cli"} {
				t.Run(component+"/"+declared.Names()[0]+"/"+scenario, func(t *testing.T) {
					name := declared.Names()[0]
					old, fresh, cliValue := "legacy", "scoped", "cli"
					if _, ok := declared.(*ucli.BoolFlag); ok {
						old, fresh, cliValue = "true", "false", "true"
					}
					t.Setenv("VINE_"+strings.TrimPrefix(keys[0], "VINE_"+component+"_"), old)
					t.Setenv(keys[0], fresh)
					want := fresh
					if scenario == "old-only" {
						require.NoError(t, os.Unsetenv(keys[0]))
						want = ""
						if flag, ok := declared.(*ucli.StringFlag); ok {
							want = flag.Value
						}
					}
					if scenario == "empty" {
						t.Setenv(keys[0], "")
						want = ""
					}
					args := []string{"serve"}
					if scenario == "cli" {
						args = append(args, "--"+name+"="+cliValue)
						want = cliValue
					}
					cmd := create()
					// Isolate the selected flag from unrelated host environment values.
					for _, flag := range cmd.Flags {
						if flag.Names()[0] == name {
							cmd.Flags = []ucli.Flag{flag}
							break
						}
					}
					var got string
					cmd.Action = func(_ context.Context, parsed *ucli.Command) error {
						switch declared.(type) {
						case *ucli.BoolFlag:
							if parsed.Bool(name) {
								got = "true"
							} else {
								got = "false"
							}
						case *ucli.StringSliceFlag, *appcli.RepeatedStringFlag:
							got = strings.Join(parsed.StringSlice(name), ",")
						default:
							got = parsed.String(name)
						}
						return nil
					}
					if _, ok := declared.(*ucli.BoolFlag); ok && want == "" {
						want = "false"
					}
					result := runCLICommand(cmd, args)
					require.Equal(t, exitCodeSuccess, result.exitCode, result.stderr)
					require.Equal(t, want, got)
				})
			}
		}
	}
}

func TestRunComponentLogging(t *testing.T) {
	oldHub, oldLink, oldPortal := startHubApp, startLinkApp, startPortalApp
	startHubApp = func(hubflag.Flag) {}
	startLinkApp = func(linkflag.Flag) {}
	startPortalApp = func(portalflag.Flag) {}
	t.Cleanup(func() { startHubApp, startLinkApp, startPortalApp = oldHub, oldLink, oldPortal })
	for _, component := range []string{"hub", "link", "portal"} {
		for _, scenario := range []string{"old-only", "scoped", "cli", "empty", "invalid"} {
			t.Run(component+"/"+scenario, func(t *testing.T) {
				logger.SetGlobalLevel(logger.LevelInfo)
				t.Cleanup(func() { logger.SetGlobalLevel(logger.LevelInfo); logger.ClearLevel("env-test") })
				prefix := "VINE_" + strings.ToUpper(component)
				t.Setenv("VINE_LOG_LEVEL", "DEBUG")
				t.Setenv("VINE_LOG_RULES", "env-test=DEBUG")
				t.Setenv(prefix+"_LOG_LEVEL", "WARN")
				t.Setenv(prefix+"_LOG_RULES", "env-test=ERROR")
				args := []string{component, "serve"}
				wantLevel, wantRule := logger.LevelWarn, logger.LevelError
				switch scenario {
				case "old-only":
					require.NoError(t, os.Unsetenv(prefix+"_LOG_LEVEL"))
					require.NoError(t, os.Unsetenv(prefix+"_LOG_RULES"))
					wantLevel, wantRule = logger.LevelInfo, logger.LevelAuto
				case "cli":
					args = append(args, "--log-level", "ERROR", "--log-rule", "env-test=INFO", "--log-rule", "env-test=DEBUG")
					wantLevel, wantRule = logger.LevelError, logger.LevelDebug
				case "empty":
					t.Setenv(prefix+"_LOG_LEVEL", "")
					t.Setenv(prefix+"_LOG_RULES", "")
					wantLevel, wantRule = logger.LevelInfo, logger.LevelAuto
				case "invalid":
					t.Setenv(prefix+"_LOG_LEVEL", "wrong")
				}
				result := run(args)
				if scenario == "invalid" {
					require.Equal(t, exitCodeError, result.exitCode)
					return
				}
				require.Equal(t, exitCodeSuccess, result.exitCode, result.stderr)
				log := logger.New("env-global-test")
				require.True(t, log.Enabled(wantLevel))
				if wantLevel != logger.LevelDebug {
					require.False(t, log.Enabled(logger.LevelDebug))
				}
				require.Equal(t, wantRule, logger.Levels()["env-test"])
			})
		}
	}
}
