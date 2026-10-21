package proxy

import (
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/core/web/spec"
	vinehttp "go.yorun.ai/vine/internal/util/httputil"
)

// ServeReverseProxy preserves response provenance and logs transport failures.
func ServeReverseProxy(proxy *httputil.ReverseProxy, ctx *gin.Context, log *logger.Logger) {
	spec.SetUpstream5xx(ctx, false)
	forwarded := vinehttp.ServeReverseProxy(proxy, _ProxyWriter{ctx.Writer}, ctx.Request, log)
	if ctx.Writer.Status() >= http.StatusInternalServerError {
		spec.SetUpstream5xx(ctx, forwarded)
	}
}

type _ProxyWriter struct{ gin.ResponseWriter }

func (w _ProxyWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w _ProxyWriter) FlushError() error {
	if flusher, ok := w.ResponseWriter.(interface{ FlushError() error }); ok {
		return flusher.FlushError()
	}
	// Gin's Flush discards the underlying error. Commit its pending headers
	// before flushing the underlying writer through ResponseController.
	if unwrapper, ok := w.ResponseWriter.(interface{ Unwrap() http.ResponseWriter }); ok {
		w.WriteHeaderNow()
		return http.NewResponseController(unwrapper.Unwrap()).Flush()
	}
	w.Flush()
	return nil
}
