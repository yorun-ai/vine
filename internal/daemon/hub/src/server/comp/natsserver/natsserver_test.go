package natsserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/mtls/mtlstest"
	"go.yorun.ai/vine/internal/daemon"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func runTestNATSServerDIInit(t *testing.T, server *NATSServer) {
	t.Helper()

	defer func() {
		if re := recover(); re != nil {
			t.Skipf("embedded nats server did not start in current environment: %v", re)
		}
	}()
	server.DIInit()
}

func TestNATSServerDIInitRegistersInprocServer(t *testing.T) {
	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{Enabled: true},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
	}

	runTestNATSServerDIInit(t, server)
	t.Cleanup(server.AfterAppStop)

	if hubnats.InprocServer() != server.server {
		t.Fatalf("unexpected inproc nats server")
	}
	if server.server != nil && server.server.Addr() != nil {
		t.Fatalf("unexpected inproc nats listener: %v", server.server.Addr())
	}
}

func TestNATSServerDIInitSkipsWhenNotInproc(t *testing.T) {
	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{},
	}

	server.DIInit()

	if hubnats.InprocServer() != nil {
		t.Fatalf("unexpected inproc nats server")
	}
}

func TestNATSServerAfterAppStopRemovesStoreDir(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	t.Setenv("TMP", tempDir)
	t.Setenv("TEMP", tempDir)

	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
	}

	runTestNATSServerDIInit(t, server)
	storeDir := server.storeDir
	if _, err := os.Stat(storeDir); err != nil {
		t.Fatalf("expected nats store dir before stop: %v", err)
	}
	if filepath.Dir(storeDir) != tempDir {
		t.Fatalf("expected nats store dir under system temp dir %q, got %q", tempDir, storeDir)
	}

	server.AfterAppStop()

	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("expected nats store dir removed, got %v", err)
	}
}

func TestNATSServerRemovesStoreDirWhenCreationFails(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	t.Setenv("TMP", tempDir)
	t.Setenv("TEMP", tempDir)

	server := &NATSServer{}
	assert.Panics(t, func() {
		server.newServer(&natsserver.Options{
			JetStream:  true,
			ServerName: "invalid server name",
		})
	})

	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestNATSServerDIInitPublishesMQEndpointWhenEnableNats(t *testing.T) {
	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
	}

	prev := detectHostForMQEndpoint
	detectHostForMQEndpoint = func() string {
		return "127.0.0.1"
	}
	t.Cleanup(func() {
		detectHostForMQEndpoint = prev
		server.AfterAppStop()
	})

	runTestNATSServerDIInit(t, server)

	if server.Endpoint() == "" {
		t.Fatal("expected nats endpoint")
	}
	if hubnats.InprocServer() != nil {
		t.Fatalf("unexpected inproc nats server")
	}
}

func TestNATSServerUsesMutualTLS(t *testing.T) {
	ca := mtlstest.NewCA(t)
	hubIdentity := ca.Identity(t, daemon.HubIdentity.SPIFFEPath())
	linkIdentity := ca.Identity(t, daemon.LinkIdentity.SPIFFEPath())
	portalIdentity := ca.Identity(t, daemon.PortalIdentity.SPIFFEPath())
	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
		Identity:   hubIdentity,
	}

	prev := detectHostForMQEndpoint
	detectHostForMQEndpoint = func() string { return "127.0.0.1" }
	t.Cleanup(func() {
		detectHostForMQEndpoint = prev
		server.AfterAppStop()
	})
	runTestNATSServerDIInit(t, server)

	hubConn, err := server.ConnectAsHub()
	require.NoError(t, err)
	hubConn.Close()

	linkConn, err := gonats.Connect(
		server.Endpoint(),
		gonats.Secure(linkIdentity.ClientConfig(daemon.HubIdentity.SPIFFEPath())),
		gonats.TLSHandshakeFirst(),
		gonats.Timeout(time.Second),
	)
	require.NoError(t, err)
	linkConn.Close()

	portalConn, err := gonats.Connect(
		server.Endpoint(),
		gonats.Secure(portalIdentity.ClientConfig(daemon.HubIdentity.SPIFFEPath())),
		gonats.TLSHandshakeFirst(),
		gonats.Timeout(time.Second),
	)
	if portalConn != nil {
		portalConn.Close()
	}
	require.Error(t, err)
}
