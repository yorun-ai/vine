package nats

import (
	"testing"
	"time"

	natsserverlib "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestConnectInproc(t *testing.T) {
	server, err := natsserverlib.NewServer(&natsserverlib.Options{
		Port:      -1,
		NoSigs:    true,
		NoLog:     true,
		JetStream: true,
		StoreDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new nats server failed: %v", err)
	}
	go server.Start()
	if !server.ReadyForConnections(2 * time.Second) {
		t.Fatalf("nats server not ready")
	}
	defer server.Shutdown()

	SetInprocServer(server)
	t.Cleanup(func() {
		SetInprocServer(nil)
	})

	conn := ConnectInproc()
	defer conn.Close()

	ch := make(chan string, 1)
	_, err = conn.Subscribe("demo.test", func(msg *nats.Msg) {
		ch <- string(msg.Data)
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if err = conn.Publish("demo.test", []byte("ok")); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if err = conn.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	select {
	case payload := <-ch:
		if payload != "ok" {
			t.Fatalf("unexpected payload: %s", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("subscribe timeout")
	}
}
