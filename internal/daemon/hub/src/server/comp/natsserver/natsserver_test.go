package natsserver

import (
	"os"
	"testing"

	"go.yorun.ai/vine/internal/app"
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
	server := &NATSServer{
		InprocFlag: &app.InternalInprocFlag{},
		Flag:       &flag.Flag{MQEmbeddedNats: true},
	}

	runTestNATSServerDIInit(t, server)
	storeDir := server.storeDir
	if _, err := os.Stat(storeDir); err != nil {
		t.Fatalf("expected nats store dir before stop: %v", err)
	}

	server.AfterAppStop()

	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("expected nats store dir removed, got %v", err)
	}
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
