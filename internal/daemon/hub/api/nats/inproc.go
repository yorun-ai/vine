package nats

import (
	natsserver "github.com/nats-io/nats-server/v2/server"
	gonats "github.com/nats-io/nats.go"
	"go.yorun.ai/vine/util/vpre"
)

var inprocServer *natsserver.Server

func SetInprocServer(server *natsserver.Server) {
	inprocServer = server
}

func InprocServer() *natsserver.Server {
	return inprocServer
}

func ConnectInproc() *gonats.Conn {
	server := InprocServer()
	vpre.CheckNotNil(server, "inproc nats server missing")
	conn, err := gonats.Connect("", gonats.InProcessServer(server))
	vpre.CheckNilError(err, "connect inproc nats failed")
	return conn
}
