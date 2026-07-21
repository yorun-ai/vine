package embedded

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	hubredis "go.yorun.ai/vine/internal/daemon/hub/api/redis"
)

func (s *Store) DialInproc(ctx context.Context) (net.Conn, error) {
	listener, ok := s.listener.(*_InprocListener)
	if !ok {
		return nil, errors.New("redis server is not inproc")
	}
	return listener.DialContext(ctx)
}

type _InprocListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
	addr  net.Addr
}

type _InprocAddr struct{}

func newInprocListener() *_InprocListener {
	return &_InprocListener{
		conns: make(chan net.Conn),
		done:  make(chan struct{}),
		addr:  _InprocAddr{},
	}
}

func (l *_InprocListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *_InprocListener) Close() error {
	l.once.Do(func() {
		close(l.done)
	})
	return nil
}

func (l *_InprocListener) Addr() net.Addr {
	return l.addr
}

func (l *_InprocListener) DialContext(ctx context.Context) (net.Conn, error) {
	serverConn, clientConn := newInprocConnPair()
	select {
	case l.conns <- serverConn:
		return clientConn, nil
	case <-ctx.Done():
		_ = serverConn.Close()
		_ = clientConn.Close()
		return nil, ctx.Err()
	case <-l.done:
		_ = serverConn.Close()
		_ = clientConn.Close()
		return nil, net.ErrClosed
	}
}

func (_InprocAddr) Network() string {
	return "redis+inproc"
}

func (_InprocAddr) String() string {
	return hubredis.RedisInprocEndpoint
}

type _InprocConn struct {
	in            *_InprocConnBuffer
	out           *_InprocConnBuffer
	local         net.Addr
	remote        net.Addr
	writeDeadline time.Time
	mutex         sync.Mutex
}

type _InprocConnBuffer struct {
	data         []byte
	closed       bool
	readDeadline time.Time
	notify       chan struct{}
	mutex        sync.Mutex
}

type _InprocTimeoutError struct{}

func newInprocConnPair() (net.Conn, net.Conn) {
	clientToServer := newInprocConnBuffer()
	serverToClient := newInprocConnBuffer()
	serverAddr := _InprocAddr{}
	clientAddr := _InprocAddr{}
	return &_InprocConn{
			in:     clientToServer,
			out:    serverToClient,
			local:  serverAddr,
			remote: clientAddr,
		}, &_InprocConn{
			in:     serverToClient,
			out:    clientToServer,
			local:  clientAddr,
			remote: serverAddr,
		}
}

func newInprocConnBuffer() *_InprocConnBuffer {
	return &_InprocConnBuffer{
		notify: make(chan struct{}),
	}
}

func (c *_InprocConn) Read(p []byte) (int, error) {
	return c.in.read(p)
}

func (c *_InprocConn) Write(p []byte) (int, error) {
	c.mutex.Lock()
	deadline := c.writeDeadline
	c.mutex.Unlock()
	if !deadline.IsZero() && !time.Now().Before(deadline) {
		return 0, _InprocTimeoutError{}
	}
	return c.out.write(p)
}

func (c *_InprocConn) Close() error {
	c.in.close()
	c.out.close()
	return nil
}

func (c *_InprocConn) LocalAddr() net.Addr {
	return c.local
}

func (c *_InprocConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *_InprocConn) SetDeadline(t time.Time) error {
	_ = c.SetReadDeadline(t)
	_ = c.SetWriteDeadline(t)
	return nil
}

func (c *_InprocConn) SetReadDeadline(t time.Time) error {
	c.in.setReadDeadline(t)
	return nil
}

func (c *_InprocConn) SetWriteDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.writeDeadline = t
	return nil
}

func (b *_InprocConnBuffer) read(p []byte) (int, error) {
	for {
		b.mutex.Lock()
		if len(b.data) > 0 {
			n := copy(p, b.data)
			b.data = b.data[n:]
			b.mutex.Unlock()
			return n, nil
		}
		if b.closed {
			b.mutex.Unlock()
			return 0, io.EOF
		}
		deadline := b.readDeadline
		notify := b.notify
		b.mutex.Unlock()

		if deadline.IsZero() {
			<-notify
			continue
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			return 0, _InprocTimeoutError{}
		}
		timer := time.NewTimer(wait)
		select {
		case <-notify:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			return 0, _InprocTimeoutError{}
		}
	}
}

func (b *_InprocConnBuffer) write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.closed {
		return 0, net.ErrClosed
	}
	b.data = append(b.data, p...)
	b.signalLocked()
	return len(p), nil
}

func (b *_InprocConnBuffer) close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	b.signalLocked()
}

func (b *_InprocConnBuffer) setReadDeadline(t time.Time) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.readDeadline = t
	b.signalLocked()
}

func (b *_InprocConnBuffer) signalLocked() {
	close(b.notify)
	b.notify = make(chan struct{})
}

func (_InprocTimeoutError) Error() string {
	return "i/o timeout"
}

func (_InprocTimeoutError) Timeout() bool {
	return true
}

func (_InprocTimeoutError) Temporary() bool {
	return true
}
