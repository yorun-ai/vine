package httputil

import (
	"errors"
	"io"
	"net/http"
	"sync"
)

func InprocRoundTrip(handler http.Handler, req *http.Request) (*http.Response, error) {
	responseCh := make(chan *http.Response, 1)
	reader, writer := io.Pipe()
	responseWriter := newResponseWriter(reader, writer, responseCh)
	go func() {
		defer responseWriter.Close()
		handler.ServeHTTP(responseWriter, req)
	}()

	select {
	case <-req.Context().Done():
		_ = writer.CloseWithError(req.Context().Err())
		return nil, req.Context().Err()
	case resp := <-responseCh:
		if resp == nil {
			return nil, errors.New("response is nil")
		}
		return resp, nil
	}
}

type _ResponseWriter struct {
	reader     *io.PipeReader
	writer     *io.PipeWriter
	responseCh chan<- *http.Response

	mutex      sync.Mutex
	header     http.Header
	statusCode int
	wrote      bool
}

func newResponseWriter(reader *io.PipeReader, writer *io.PipeWriter, responseCh chan<- *http.Response) *_ResponseWriter {
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
	w.responseCh <- &http.Response{
		StatusCode: statusCode,
		Header:     w.header.Clone(),
		Body:       w.reader,
	}
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
