package natsserver

import (
	"fmt"
	"net"
	"os"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/logger"
	hubnats "go.yorun.ai/vine/internal/daemon/hub/api/nats"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
	"go.yorun.ai/vine/util/vnet"
	"go.yorun.ai/vine/util/vpre"
)

const natsServerReadyTimeout = 5 * time.Second

const natsStoreDir = "/tmp"

var detectHostForMQEndpoint = func() string {
	return vnet.DetectHostIP()
}

type NATSServer struct {
	app.BaseComponent

	InprocFlag *app.InternalInprocFlag `inject:""`
	Flag       *flag.Flag              `inject:""`

	server   *natsserver.Server
	storeDir string
	endpoint string
}

func (s *NATSServer) DIInit() {
	if !s.Flag.MQEmbeddedNats {
		return
	}

	options := s.serverOptions(s.InprocFlag.Enabled)
	s.server, s.storeDir = s.newServer(options)

	if s.InprocFlag.Enabled {
		hubnats.SetInprocServer(s.server)
		logger.Info("vine.hub nats server started", "mode", "inproc")
		return
	}

	addr, ok := s.server.Addr().(*net.TCPAddr)
	vpre.Check(ok, "nats server addr is not tcp")
	s.endpoint = fmt.Sprintf("nats://%s:%d", detectHostForMQEndpoint(), addr.Port)
	logger.Info("vine.hub nats server started", "mode", "remote", "addr", addr.String(), "endpoint", s.endpoint)
}

func (s *NATSServer) Endpoint() string {
	return s.endpoint
}

func (s *NATSServer) Port() int {
	addr, ok := s.server.Addr().(*net.TCPAddr)
	vpre.Check(ok, "nats server addr is not tcp")
	return addr.Port
}

func (s *NATSServer) AfterAppStop() {
	if s.server == nil {
		return
	}
	if hubnats.InprocServer() == s.server {
		hubnats.SetInprocServer(nil)
	}
	s.server.Shutdown()
	s.server.WaitForShutdown()
	// Cleanup failure on the temporary JetStream dir should not affect app shutdown.
	_ = os.RemoveAll(s.storeDir)
}

func (s *NATSServer) serverOptions(isInproc bool) *natsserver.Options {
	options := &natsserver.Options{
		NoSigs:    true,
		NoLog:     true,
		JetStream: true,
	}
	if isInproc {
		options.DontListen = true
		return options
	}
	options.Port = natsserver.RANDOM_PORT
	return options
}

func (s *NATSServer) newServer(options *natsserver.Options) (*natsserver.Server, string) {
	storeDir, err := os.MkdirTemp(natsStoreDir, "vine-nats-jetstream-*")
	vpre.CheckNilError(err, "create nats jetstream dir failed")
	options.StoreDir = storeDir
	server, err := natsserver.NewServer(options)
	vpre.CheckNilError(err, "create nats server failed")

	go server.Start()
	vpre.Check(server.ReadyForConnections(natsServerReadyTimeout), "nats server start failed")
	return server, storeDir
}
