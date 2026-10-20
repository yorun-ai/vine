package httputil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
)

func InprocRoundTrip(handler http.Handler, req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	// Match a real HTTP server's request lifetime and execution boundary. ReverseProxy
	// relies on ServerContextKey to abort, rather than suppress, body copy failures.
	ctx, cancel := context.WithCancel(req.Context())
	if ctx.Value(http.ServerContextKey) == nil {
		ctx = context.WithValue(ctx, http.ServerContextKey, new(http.Server{Handler: handler}))
	}
	req = req.Clone(ctx)
	responseCh := make(chan _InprocResponse, 1)
	reader, writer := io.Pipe()
	responseWriter := newResponseWriter(reader, writer, responseCh)
	responseWriter.request = req
	responseWriter.reader = &_InprocBody{ReadCloser: reader, cancel: cancel}
	go func() {
		stop := context.AfterFunc(req.Context(), func() { _ = writer.CloseWithError(req.Context().Err()) })
		defer stop()
		defer func() {
			if recovered := recover(); recovered != nil {
				if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					responseWriter.closeWithError(io.ErrUnexpectedEOF)
					return
				}
				panic(recovered)
			}
			responseWriter.Close()
		}()
		handler.ServeHTTP(responseWriter, req)
	}()

	select {
	case <-req.Context().Done():
		cancel()
		_ = writer.CloseWithError(req.Context().Err())
		return nil, req.Context().Err()
	case result := <-responseCh:
		resp := result.response
		if result.err != nil {
			cancel()
			return nil, result.err
		}
		if err := req.Context().Err(); err != nil {
			cancel()
			_ = writer.CloseWithError(err)
			return nil, err
		}
		if resp == nil {
			cancel()
			return nil, errors.New("response is nil")
		}
		return resp, nil
	}
}

type _InprocResponse struct {
	response *http.Response
	err      error
}

type _ResponseWriter struct {
	request    *http.Request
	reader     io.ReadCloser
	writer     *io.PipeWriter
	responseCh chan<- _InprocResponse

	mutex      sync.Mutex
	header     http.Header
	statusCode int
	wrote      bool
}

func newResponseWriter(reader *io.PipeReader, writer *io.PipeWriter, responseCh chan<- _InprocResponse) *_ResponseWriter {
	return &_ResponseWriter{
		reader:     reader,
		writer:     writer,
		responseCh: responseCh,
		header:     http.Header{},
	}
}

func (w *_ResponseWriter) Header() http.Header {
	return w.header
}

func (w *_ResponseWriter) WriteHeader(statusCode int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.wrote {
		return
	}
	w.wrote = true
	w.statusCode = statusCode
	length := int64(-1)
	if value := w.header.Get("Content-Length"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed >= 0 {
			length = parsed
		}
	}
	w.responseCh <- _InprocResponse{response: &http.Response{
		Request:       w.request,
		ContentLength: length,
		StatusCode:    statusCode,
		Header:        w.header.Clone(),
		Body:          w.reader,
	}}
}

func (w *_ResponseWriter) Write(data []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.writer.Write(data)
}

func (w *_ResponseWriter) Flush() {
	w.WriteHeader(http.StatusOK)
}

func (w *_ResponseWriter) Close() {
	w.WriteHeader(http.StatusOK)
	_ = w.writer.Close()
}

func (w *_ResponseWriter) closeWithError(err error) {
	w.mutex.Lock()
	if !w.wrote {
		w.wrote = true
		w.responseCh <- _InprocResponse{err: err}
	}
	w.mutex.Unlock()
	_ = w.writer.CloseWithError(err)
}

type _InprocBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *_InprocBody) Close() error {
	b.cancel()
	return b.ReadCloser.Close()
}
