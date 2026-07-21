package proxy

import (
	"context"
	"net"
	stdhttputil "net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.yorun.ai/vine/internal/core/logger"
	"go.yorun.ai/vine/internal/util/httputil"
)

const (
	defaultDialTimeout       = 100 * time.Millisecond
	defaultDetectionInterval = time.Second
)

type Option struct {
	Target            *url.URL
	DialTimeout       time.Duration
	DetectionInterval time.Duration
}

type ReverseProxy struct {
	target         *url.URL
	dialTimeout    time.Duration
	detectInterval time.Duration
	reverseProxy   *stdhttputil.ReverseProxy
	context        context.Context
	cancel         context.CancelFunc

	available atomic.Bool
}

type _RequestPathContextKey struct{}

func NewReverseProxy(opt Option) *ReverseProxy {
	if opt.DialTimeout == 0 {
		opt.DialTimeout = defaultDialTimeout
	}
	if opt.DetectionInterval == 0 {
		opt.DetectionInterval = defaultDetectionInterval
	}

	proxy := &ReverseProxy{
		target:         opt.Target,
		dialTimeout:    opt.DialTimeout,
		detectInterval: opt.DetectionInterval,
	}
	proxy.context, proxy.cancel = context.WithCancel(context.Background())
	proxy.init()
	return proxy
}

func (p *ReverseProxy) init() {
	p.reverseProxy = &stdhttputil.ReverseProxy{
		Rewrite: func(request *stdhttputil.ProxyRequest) {
			requestPath := request.In.Context().Value(_RequestPathContextKey{}).(string)
			request.SetURL(p.target)
			request.Out.Host = p.target.Host
			request.Out.URL.Path = requestPath
			request.Out.URL.RawPath = ""
		},
		Transport: httputil.NewUpgradeIdleTransport(nil, httputil.DefaultStreamIdleTimeout),
	}
	p.refreshAvailable()
	go p.watch()
}

func (p *ReverseProxy) Serve(ginCtx *gin.Context) bool {
	if !p.available.Load() {
		return false
	}

	logger.Debug("reverse proxy request",
		"method", ginCtx.Request.Method,
		"path", ginCtx.Request.URL.RequestURI(),
		"target", p.target.String(),
	)

	requestPath := ginCtx.Param("path")
	requestCtx := context.WithValue(ginCtx.Request.Context(), _RequestPathContextKey{}, requestPath)
	request := ginCtx.Request.WithContext(requestCtx)
	p.reverseProxy.ServeHTTP(ginCtx.Writer, request)
	return true
}

func (p *ReverseProxy) watch() {
	ticker := time.NewTicker(p.detectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.context.Done():
			return
		case <-ticker.C:
			p.refreshAvailable()
		}
	}
}

func (p *ReverseProxy) Close() {
	p.cancel()
}

func (p *ReverseProxy) refreshAvailable() {
	p.available.Store(p.detectAvailable())
}

func (p *ReverseProxy) detectAvailable() bool {
	conn, err := net.DialTimeout("tcp", p.target.Host, p.dialTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
