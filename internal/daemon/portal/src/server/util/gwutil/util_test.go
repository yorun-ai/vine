package gwutil

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForwardDeadlinePreservesNetworkOperation(t *testing.T) {
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	dialError := &net.OpError{Op: "dial", Err: context.DeadlineExceeded}
	err := normalizeForwardError(ctx, dialError)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var operation *net.OpError
	require.ErrorAs(t, err, &operation)
	require.Same(t, dialError, operation)
}
