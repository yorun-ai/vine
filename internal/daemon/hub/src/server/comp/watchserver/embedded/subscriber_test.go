package embedded

import (
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/redcon"
)

type stalledSubscriber struct {
	redcon.DetachedConn
	conn net.Conn
}

func (c *stalledSubscriber) NetConn() net.Conn { return c.conn }
func (c *stalledSubscriber) Flush() error      { _, err := c.conn.Write([]byte("notification")); return err }

func TestSubscriberDisconnectsWhenWriteTimesOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		server, client := net.Pipe()
		defer server.Close()
		defer client.Close()
		conn := &_SubscriberDetachedConn{DetachedConn: &stalledSubscriber{conn: server}}
		started := time.Now()
		err := conn.Flush()
		var timeout net.Error
		require.True(t, errors.As(err, &timeout))
		require.True(t, timeout.Timeout())
		require.Equal(t, subscriberWriteTimeout, time.Since(started))
		// Closing the failed connection makes the subscriber reconnect and reload.
		_, err = client.Read(make([]byte, 1))
		require.Error(t, err)
	})
}
