package httputil

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestReadStreamBodyTimesOutWhenIdle(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()

	buffer := make([]byte, 8)
	_, err := readStreamBody(reader, buffer, time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected error: %v", err)
	}
}
