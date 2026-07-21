package spec

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"

	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/util/vpre"
	"go.yorun.ai/vine/util/vslice"
)

const (
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderAccessControlReqHeaders       = "Access-Control-Request-Headers"

	HeaderPortalTraceId = "portal-trace-id"
)

var corsExposeHeaders = []string{
	rpchttp.HeaderRpcStatus,
	rpchttp.HeaderRpcServer,
}

func ServeOptions(w http.ResponseWriter, r *http.Request, cors redised.PortalCors, entryOrigin EntryOrigin, allowedMethods []string) {
	if ApplyCORS(w, r, cors, entryOrigin) {
		methods := append(vslice.Clone(allowedMethods), http.MethodOptions)
		header := w.Header()
		header.Set(HeaderAccessControlAllowMethods, strings.Join(methods, ", "))
		header.Set(HeaderAccessControlMaxAge, "600")

		if requestHeaders := r.Header.Get(HeaderAccessControlReqHeaders); requestHeaders != "" {
			header.Set(HeaderAccessControlAllowHeaders, requestHeaders)
			header.Add("Vary", HeaderAccessControlReqHeaders)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func ApplyCORS(w http.ResponseWriter, r *http.Request, cors redised.PortalCors, entryOrigin EntryOrigin) bool {
	origin := r.Header.Get("Origin")
	if !isCORSRequest(origin) || !allowOrigin(cors, origin, entryOrigin) {
		return false
	}

	header := w.Header()
	header.Set(HeaderAccessControlAllowOrigin, origin)
	header.Set(HeaderAccessControlAllowCredentials, "true")
	header.Set(HeaderAccessControlExposeHeaders, strings.Join(corsExposeHeaders, ", "))
	header.Add("Vary", "Origin")
	return true
}

func ExposeHeader(header http.Header, key string) {
	value := header.Get(HeaderAccessControlExposeHeaders)
	if value == "" {
		header.Set(HeaderAccessControlExposeHeaders, key)
		return
	}
	for _, item := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(item), key) {
			return
		}
	}
	header.Set(HeaderAccessControlExposeHeaders, value+", "+key)
}

// isCORSRequest reports whether the request carries browser CORS semantics.
// Same-origin and non-browser requests usually omit Origin and do not need CORS response headers.
func isCORSRequest(origin string) bool {
	return origin != ""
}

func allowOrigin(cors redised.PortalCors, origin string, entryOrigin EntryOrigin) bool {
	switch cors.Mode {
	case redised.PortalCorsModeDisabled:
		return false
	case redised.PortalCorsModeSameDomain:
		return sameDomainOrigin(origin, entryOrigin)
	case redised.PortalCorsModeStrict:
		return vslice.Any(cors.AllowedOrigins, func(allowed string) bool {
			return sameOrigin(allowed, origin)
		})
	default:
		vpre.MustNotReach()
		return false
	}
}

func sameOrigin(left string, right string) bool {
	leftOrigin, ok := parseOrigin(left)
	if !ok {
		return false
	}
	rightOrigin, ok := parseOrigin(right)
	if !ok {
		return false
	}
	return leftOrigin == rightOrigin
}

func sameDomainOrigin(origin string, entryOrigin EntryOrigin) bool {
	parsedOrigin, ok := parseOrigin(origin)
	if !ok {
		return false
	}
	parsedEntryOrigin := parseEntryOrigin(entryOrigin)
	if parsedOrigin == parsedEntryOrigin {
		return true
	}

	originDomain, ok := effectiveTLDPlusOne(parsedOrigin.host)
	if !ok {
		return false
	}
	ruleDomain, ok := effectiveTLDPlusOne(parsedEntryOrigin.host)
	if !ok {
		return false
	}
	return strings.EqualFold(originDomain, ruleDomain)
}

type _Origin struct {
	scheme string
	host   string
	port   string
}

func parseOrigin(origin string) (_Origin, bool) {
	parsed, err := url.Parse(origin)
	if err != nil ||
		parsed.Scheme == "" ||
		parsed.Host == "" ||
		parsed.Path != "" ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return _Origin{}, false
	}

	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case string(SchemeHTTP):
			port = "80"
		case string(SchemeHTTPS):
			port = "443"
		default:
			return _Origin{}, false
		}
	}

	return _Origin{
		scheme: strings.ToLower(parsed.Scheme),
		host:   strings.ToLower(parsed.Hostname()),
		port:   port,
	}, true
}

func parseEntryOrigin(origin EntryOrigin) _Origin {
	port := origin.Port
	if port == 0 {
		port = 80
		if origin.Scheme == SchemeHTTPS {
			port = 443
		}
	}

	return _Origin{
		scheme: strings.ToLower(string(origin.Scheme)),
		host:   strings.ToLower(hostname(origin.Host)),
		port:   strconv.Itoa(port),
	}
}

func effectiveTLDPlusOne(host string) (string, bool) {
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return "", false
	}
	return domain, true
}

func hostname(host string) string {
	ret, _, err := net.SplitHostPort(host)
	if err == nil {
		return ret
	}
	return host
}
