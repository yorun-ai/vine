package http

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestReadBodyAcceptsBodyAtLimit(t *testing.T) {
	body, err := readBody(strings.NewReader("12345678"), -1, 8, "request")
	if err != nil {
		t.Fatalf("readBody() error = %v", err)
	}
	if string(body) != "12345678" {
		t.Fatalf("readBody() = %q", body)
	}
}

func TestReadBodyRejectsUnknownLengthBodyOverLimit(t *testing.T) {
	_, err := readBody(strings.NewReader("123456789"), -1, 8, "request")
	if err == nil || err.Error() != "rpc request body exceeds 8 byte limit" {
		t.Fatalf("readBody() error = %v", err)
	}
}

func TestReadBodyRejectsDeclaredLengthBeforeReading(t *testing.T) {
	reader := &failReader{}
	_, err := readBody(reader, 9, 8, "response")
	if err == nil || err.Error() != "rpc response body exceeds 8 byte limit" {
		t.Fatalf("readBody() error = %v", err)
	}
	if reader.read {
		t.Fatal("readBody() read a body with an oversized declared length")
	}
}

func TestReadRequestAndResponseBodyLimits(t *testing.T) {
	request := &http.Request{
		Body:          http.NoBody,
		ContentLength: MaxRequestBodyBytes + 1,
	}
	if _, err := ReadRequestBody(request); err == nil {
		t.Fatal("ReadRequestBody() error = nil")
	}

	response := &http.Response{
		Body:          http.NoBody,
		ContentLength: MaxResponseBodyBytes + 1,
	}
	if _, err := ReadResponseBody(response); err == nil {
		t.Fatal("ReadResponseBody() error = nil")
	}
}

func TestReadBodyReturnsReadError(t *testing.T) {
	want := errors.New("read failed")
	_, err := readBody(&errorReader{err: want}, -1, 8, "request")
	if !errors.Is(err, want) {
		t.Fatalf("readBody() error = %v, want %v", err, want)
	}
}

type failReader struct {
	read bool
}

func (r *failReader) Read([]byte) (int, error) {
	r.read = true
	return 0, errors.New("unexpected read")
}

type errorReader struct {
	err error
}

func (r *errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
