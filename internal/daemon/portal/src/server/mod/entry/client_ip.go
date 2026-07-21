package entry

import (
	"net"
	"net/http"
	"strings"
)

const (
	headerForwardedFor = "X-Forwarded-For"
	headerRealIP       = "X-Real-IP"
)

func clientIP(r *http.Request) string {
	if ip := firstForwardedIP(r.Header.Get(headerForwardedFor)); ip != "" {
		return ip
	}
	if ip := parseIP(r.Header.Get(headerRealIP)); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return parseIP(r.RemoteAddr)
}

func firstForwardedIP(value string) string {
	for item := range strings.SplitSeq(value, ",") {
		if ip := parseIP(item); ip != "" {
			return ip
		}
	}
	return ""
}

func parseIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}
