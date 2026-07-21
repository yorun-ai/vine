package embedded

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInprocListenerAcceptUnblocksOnClose(t *testing.T) {
	listener := newInprocListener()
	done := make(chan error, 1)
	go func() {
		_, err := listener.Accept()
		done <- err
	}()

	require.NoError(t, listener.Close())

	select {
	case err := <-done:
		assert.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(time.Second):
		t.Fatal("accept did not unblock")
	}
}

func TestInprocListenerDialHonorsContextCancellation(t *testing.T) {
	listener := newInprocListener()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := listener.DialContext(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestInprocListenerDialFailsAfterClose(t *testing.T) {
	listener := newInprocListener()
	require.NoError(t, listener.Close())

	_, err := listener.DialContext(context.Background())

	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestInprocConnBuffersWrites(t *testing.T) {
	serverConn, clientConn := newInprocConnPair()
	defer serverConn.Close()
	defer clientConn.Close()

	n, err := clientConn.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	buf := make([]byte, 8)
	n, err = serverConn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf[:n]))
}

func TestInprocConnReadDeadline(t *testing.T) {
	serverConn, clientConn := newInprocConnPair()
	defer serverConn.Close()
	defer clientConn.Close()

	require.NoError(t, serverConn.SetReadDeadline(time.Now().Add(10*time.Millisecond)))
	buf := make([]byte, 1)
	_, err := serverConn.Read(buf)

	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	assert.True(t, netErr.Timeout())
}

func TestInprocConnCloseUnblocksReadAndRejectsWrite(t *testing.T) {
	serverConn, clientConn := newInprocConnPair()
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		_, err := serverConn.Read(buf)
		done <- err
	}()

	require.NoError(t, clientConn.Close())

	select {
	case err := <-done:
		assert.True(t, errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed))
	case <-time.After(time.Second):
		t.Fatal("read did not unblock")
	}

	_, err := serverConn.Write([]byte("x"))
	assert.ErrorIs(t, err, net.ErrClosed)
}
