package core

import (
	"fmt"
	"strings"

	"go.yorun.ai/vine/internal/core/ex"
)

// PortalEntryHTTP configures both HTTP transports of one logical entry.
type PortalEntryHTTP struct {
	HttpEnabled  bool `json:"httpEnabled"`
	HttpPort     int  `json:"httpPort"`
	HttpsEnabled bool `json:"httpsEnabled"`
	HttpsPort    int  `json:"httpsPort"`
	AutoHTTPS    bool `json:"autoHTTPS"`
}

// PortalEntryHTTPUpdate changes only the HTTP fields supplied by the caller.
type PortalEntryHTTPUpdate struct {
	HttpEnabled  *bool
	HttpPort     *int
	HttpsEnabled *bool
	HttpsPort    *int
	AutoHTTPS    *bool
}

// DefaultPortalEntryHTTP returns the defaults for the protocol-based vocabulary.
func DefaultPortalEntryHTTP() PortalEntryHTTP {
	return PortalEntryHTTP{HttpEnabled: true, HttpPort: 80, HttpsEnabled: true, HttpsPort: 443, AutoHTTPS: true}
}

// Apply merges the supplied fields into an owned configuration value.
func (u PortalEntryHTTPUpdate) Apply(value PortalEntryHTTP) PortalEntryHTTP {
	if u.HttpEnabled != nil {
		value.HttpEnabled = *u.HttpEnabled
	}
	if u.HttpPort != nil {
		value.HttpPort = *u.HttpPort
	}
	if u.HttpsEnabled != nil {
		value.HttpsEnabled = *u.HttpsEnabled
	}
	if u.HttpsPort != nil {
		value.HttpsPort = *u.HttpsPort
	}
	if u.AutoHTTPS != nil {
		value.AutoHTTPS = *u.AutoHTTPS
	}
	return value
}

// PortalEntryAccess is one enabled transport of an entry.
type PortalEntryAccess struct {
	Scheme string
	Port   int
}

// Accesses returns the enabled transports, including the automatic redirect ingress.
func (e PortalEntry) Accesses() []PortalEntryAccess {
	e = normalizePortalEntry(e)
	accesses := []PortalEntryAccess{}
	if e.Http.HttpEnabled {
		accesses = append(accesses, PortalEntryAccess{Scheme: "http", Port: e.Http.HttpPort})
	}
	if e.Http.HttpsEnabled {
		accesses = append(accesses, PortalEntryAccess{Scheme: "https", Port: e.Http.HttpsPort})
	}
	return accesses
}

// NormalizePortalEntry converts legacy entries and validates the canonical configuration.
func NormalizePortalEntry(entry PortalEntry) PortalEntry { return normalizePortalEntry(entry) }

func normalizePortalEntryHTTP(entry PortalEntry) PortalEntry {
	entry.Protocol = strings.ToLower(strings.TrimSpace(entry.Protocol))
	if entry.Protocol == "" {
		entry.Scheme = strings.ToLower(strings.TrimSpace(entry.Scheme))
		port := portalEntrySchemePort(entry.Scheme, entry.Port)
		entry.Protocol = "http"
		entry.Http = new(PortalEntryHTTP{HttpEnabled: entry.Scheme == "http", HttpPort: 80, HttpsEnabled: entry.Scheme == "https", HttpsPort: 443})
		if entry.Scheme == "http" {
			entry.Http.HttpPort = port
		} else {
			entry.Http.HttpsPort = port
		}
	} else {
		ex.PanicNewIfNot(entry.Protocol == "http", ex.OperationFailed, ex.F("unknown portal entry protocol: %s", entry.Protocol))
		if entry.Http == nil {
			entry.Http = new(DefaultPortalEntryHTTP())
		} else {
			entry.Http = new(*entry.Http)
		}
	}
	config := entry.Http
	ex.PanicNewIfNot(config.HttpPort >= 0 && config.HttpPort <= 65535 && config.HttpsPort >= 0 && config.HttpsPort <= 65535, ex.OperationFailed, "portal entry port must be between 0 and 65535")
	if config.HttpPort == 0 {
		config.HttpPort = 80
	}
	if config.HttpsPort == 0 {
		config.HttpsPort = 443
	}
	ex.PanicNewIfNot(config.HttpEnabled || config.HttpsEnabled, ex.OperationFailed, "portal entry must enable HTTP or HTTPS")
	ex.PanicNewIfNot(!config.AutoHTTPS || (config.HttpEnabled && config.HttpsEnabled), ex.OperationFailed, "autoHTTPS requires HTTP and HTTPS to be enabled")
	ex.PanicNewIfNot(!config.HttpEnabled || !config.HttpsEnabled || config.HttpPort != config.HttpsPort, ex.OperationFailed, "HTTP and HTTPS cannot listen on the same port")
	// The deprecated projection remains available for single-transport API clients.
	entry.Scheme, entry.Port = "", 0
	if config.HttpEnabled && !config.HttpsEnabled {
		entry.Scheme, entry.Port = "http", config.HttpPort
	}
	if config.HttpsEnabled && !config.HttpEnabled {
		entry.Scheme, entry.Port = "https", config.HttpsPort
	}
	return entry
}

func portalEntryAccessText(entry PortalEntry) string {
	addresses := []string{}
	for _, access := range entry.Accesses() {
		addresses = append(addresses, fmt.Sprintf("%s://%s:%d", access.Scheme, entry.Host, access.Port))
	}
	return strings.Join(addresses, ", ")
}
