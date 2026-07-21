package vnet

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type _TestConn struct{}

func (*_TestConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (*_TestConn) Write([]byte) (int, error)        { return 0, nil }
func (*_TestConn) Close() error                     { return nil }
func (*_TestConn) LocalAddr() net.Addr              { return &net.IPAddr{} }
func (*_TestConn) RemoteAddr() net.Addr             { return &net.IPAddr{} }
func (*_TestConn) SetDeadline(time.Time) error      { return nil }
func (*_TestConn) SetReadDeadline(time.Time) error  { return nil }
func (*_TestConn) SetWriteDeadline(time.Time) error { return nil }

type _TestUDPConn struct {
	_TestConn

	localAddr net.Addr
}

func (c *_TestUDPConn) LocalAddr() net.Addr {
	return c.localAddr
}

func resetDetectHostIPForTest() {
	hostIPOnce = sync.Once{}
	hostIP = ""
}

func TestDetectHostIPRejectsNonUDPAddr(t *testing.T) {
	prev := dialUDP
	dialUDP = func(string) (net.Conn, error) {
		return &_TestConn{}, nil
	}
	t.Cleanup(func() {
		dialUDP = prev
	})
	resetDetectHostIPForTest()

	assert.PanicsWithError(t, "detect host ip local addr is not udp", func() {
		detectHostIP(defaultProbeAddress)
	})
}

func TestDetectHostIPCachesResult(t *testing.T) {
	callCount := 0
	prev := dialUDP
	dialUDP = func(string) (net.Conn, error) {
		callCount++
		return &_TestUDPConn{
			localAddr: &net.UDPAddr{IP: net.ParseIP("10.0.0.8"), Port: 12345},
		}, nil
	}
	t.Cleanup(func() {
		dialUDP = prev
	})
	resetDetectHostIPForTest()

	assert.Equal(t, "10.0.0.8", DetectHostIP())
	assert.Equal(t, "10.0.0.8", DetectHostIP())
	assert.Equal(t, 1, callCount)
}
