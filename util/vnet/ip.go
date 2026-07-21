package vnet

import (
	"net"
	"sync"

	"go.yorun.ai/vine/util/vpre"
)

const defaultProbeAddress = "8.8.8.8:80"

// dialUDP is overridden in tests to avoid relying on real network access.
var dialUDP = func(address string) (net.Conn, error) {
	return net.Dial("udp", address)
}

var (
	hostIPOnce sync.Once
	hostIP     string
)

// DetectHostIP returns the cached local IP selected for the default outbound probe address.
func DetectHostIP() string {
	hostIPOnce.Do(func() {
		hostIP = detectHostIP(defaultProbeAddress)
	})
	return hostIP
}

// DetectHostIPByProbeAddress returns the local IP the kernel selects to reach address.
// The UDP probe does not send application data.
func DetectHostIPByProbeAddress(address string) string {
	return detectHostIP(address)
}

func detectHostIP(address string) string {
	// Use a UDP dial only to let the kernel pick the outbound local IP that would
	// be used toward the fixed probe address. We never write application data on this socket.
	conn, err := dialUDP(address)
	vpre.CheckNilError(err, "detect host ip failed")
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	vpre.Check(ok, "detect host ip local addr is not udp")

	ip := localAddr.IP.String()
	vpre.Check(ip != "", "detect host ip is empty")
	return ip
}
