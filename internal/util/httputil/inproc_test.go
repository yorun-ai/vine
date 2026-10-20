package httputil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInprocRoundTripAbort(t *testing.T) {
	for _, committed := range []bool{false, true} {
		response, err := InprocRoundTrip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if committed {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("partial"))
			}
			panic(http.ErrAbortHandler)
		}), httptest.NewRequest("GET", "/", nil))
		if !committed {
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("uncommitted abort: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if string(data) != "partial" || !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("body=%q err=%v", data, err)
		}
	}
}

func TestInprocRoundTripCancellationUnblocksBodyWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	response, err := InprocRoundTrip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("blocked"))
	}), httptest.NewRequest("GET", "/", nil).WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not exit")
	}
	if _, err := io.ReadAll(response.Body); !errors.Is(err, context.Canceled) {
		t.Fatalf("body error: %v", err)
	}
}

func TestInprocBodyCloseReleasesBlockedHandler(t *testing.T) {
	done := make(chan struct{})
	request := httptest.NewRequest("GET", "/", nil)
	response, err := InprocRoundTrip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("blocked"))
		<-r.Context().Done()
	}), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("body close did not release handler")
	}
	if request.Context().Err() != nil {
		t.Fatal("body close cancelled parent request")
	}
}
