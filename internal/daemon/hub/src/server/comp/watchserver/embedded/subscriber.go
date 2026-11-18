package embedded

import (
	"time"

	"github.com/tidwall/redcon"
)

const subscriberWriteTimeout = 5 * time.Second

// Redcon ignores PubSub Flush errors. Close failed writers so the client must
// reconnect and reconcile instead of silently missing subsequent notifications.
type _SubscriberConn struct {
	redcon.Conn
}

func (c *_SubscriberConn) Detach() redcon.DetachedConn {
	return new(_SubscriberDetachedConn{DetachedConn: c.Conn.Detach()})
}

type _SubscriberDetachedConn struct {
	redcon.DetachedConn
}

func (c *_SubscriberDetachedConn) Flush() error {
	conn := c.NetConn()
	if err := conn.SetWriteDeadline(time.Now().Add(subscriberWriteTimeout)); err != nil {
		_ = conn.Close()
		return err
	}
	err := c.DetachedConn.Flush()
	if err != nil {
		_ = conn.Close()
	}
	return err
}
