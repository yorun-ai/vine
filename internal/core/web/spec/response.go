package spec

import "github.com/gin-gonic/gin"

type _Upstream5xxKey struct{}

// SetUpstream5xx marks a forwarded upstream 5xx for the request completion log.
// Clearing an absent marker leaves ordinary requests free of tracking state.
func SetUpstream5xx(ctx *gin.Context, upstream5xx bool) {
	if upstream5xx || IsUpstream5xx(ctx) {
		ctx.Set(_Upstream5xxKey{}, upstream5xx)
	}
}

// IsUpstream5xx reports whether this response is a forwarded upstream 5xx.
// The marker is reset with Gin's request scope and is never propagated to another hop.
func IsUpstream5xx(ctx *gin.Context) bool { return ctx.GetBool(_Upstream5xxKey{}) }
