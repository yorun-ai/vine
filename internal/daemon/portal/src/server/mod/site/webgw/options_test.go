package webgw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	webspec "go.yorun.ai/vine/internal/core/web/spec"
)

func TestSiteTimeoutAndClientBudget(t *testing.T) {
	for _, test := range []struct {
		name    string
		total   time.Duration
		options string
		want    time.Duration
		invalid bool
	}{
		{"legacy default", 0, "", 30 * time.Second, false},
		{"legacy override", 0, "timeout=120s", 120 * time.Second, false},
		{"legacy limit", 0, "timeout=121s", 0, true},
		{"site extended", 15 * time.Minute, "", 15 * time.Minute, false},
		{"client shortens", 15 * time.Minute, "timeout=5m", 5 * time.Minute, false},
		{"client capped", 15 * time.Minute, "timeout=1h", 15 * time.Minute, false},
		{"site shortens", 5 * time.Second, "timeout=30s", 5 * time.Second, false},
		{"invalid still rejected", time.Minute, "timeout=oops", 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancelClient := context.WithCancel(t.Context())
			cancelClient()
			request := httptest.NewRequest("GET", "http://example.com/go/a", nil).WithContext(ctx)
			if test.options != "" {
				request.Header.Set(webspec.HeaderWebOptions, test.options)
			}
			forwarded, cancel, err := requestWithWebOptionsTimeout(request, t.Context(), test.total)
			if test.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer cancel()
			require.NoError(t, forwarded.Context().Err())
			deadline, ok := forwarded.Context().Deadline()
			require.True(t, ok)
			require.InDelta(t, test.want.Seconds(), time.Until(deadline).Seconds(), 0.2)
			options, err := webspec.DecodeOptionsFromHeader(forwarded.Header)
			require.NoError(t, err)
			require.Equal(t, test.want, options.Timeout)
		})
	}
}

func TestExplicitSiteTimeoutBoundsStreams(t *testing.T) {
	for _, stream := range []string{"sse", "websocket"} {
		t.Run(stream, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://example.com/events", nil)
			if stream == "sse" {
				request.Header.Set("Accept", "text/event-stream")
			} else {
				request.Header.Set("Connection", "Upgrade")
				request.Header.Set("Upgrade", "websocket")
			}
			forwarded, cancel, err := requestWithWebOptionsTimeout(request, t.Context(), 50*time.Millisecond)
			require.NoError(t, err)
			defer cancel()
			select {
			case <-forwarded.Context().Done():
				require.ErrorIs(t, forwarded.Context().Err(), context.DeadlineExceeded)
			case <-time.After(time.Second):
				t.Fatal("stream total timeout not applied")
			}
		})
	}
}
