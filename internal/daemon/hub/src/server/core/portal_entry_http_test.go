package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/util/vcode"
)

func TestPortalHTTPEffective(t *testing.T) {
	for _, tc := range []struct {
		name   string
		raw    PortalEntryHTTP
		cert   bool
		http   bool
		https  bool
		auto   bool
		reason string
	}{
		{"empty", PortalEntryHTTP{}, false, true, false, false, "NO_DOMAIN"},
		{"IP", PortalEntryHTTP{Host: "127.0.0.1"}, true, true, false, false, "NO_DOMAIN"},
		{"domain without cert", PortalEntryHTTP{Host: "example.com"}, false, true, false, false, "NO_MATCHING_CERTIFICATE"},
		{"automatic", PortalEntryHTTP{Host: "example.com"}, true, true, true, true, "ENABLED"},
		{"explicit HTTPS", PortalEntryHTTP{Host: "example.com", Https: "ENABLED"}, true, true, true, true, "ENABLED"},
		{"HTTP only", PortalEntryHTTP{Host: "example.com", Https: "DISABLED"}, true, true, false, false, "DISABLED"},
		{"HTTPS only", PortalEntryHTTP{Host: "example.com", Http: "DISABLED"}, true, false, true, false, "ENABLED"},
		{"pending HTTPS", PortalEntryHTTP{Host: "example.com", Http: "DISABLED"}, false, false, false, false, "NO_MATCHING_CERTIFICATE"},
		{"no redirect", PortalEntryHTTP{Host: "example.com", AutoHTTPS: "DISABLED"}, true, true, true, false, "ENABLED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &tc.raw})
			got := entry.Http.Effective(tc.cert)
			require.Equal(t, tc.http, got.HttpEnabled)
			require.Equal(t, tc.https, got.HttpsEnabled)
			require.Equal(t, tc.auto, got.AutoHTTPSEnabled)
			require.Equal(t, tc.reason, got.HTTPSReason)
			require.Equal(t, 80, got.HttpPort)
			require.Equal(t, 443, got.HttpsPort)
			require.Equal(t, tc.raw.Http.normalized("http"), entry.Http.Http)
			require.Equal(t, tc.raw.Https.normalized("https"), entry.Http.Https)
			require.Equal(t, tc.raw.AutoHTTPS.normalized("autohttps"), entry.Http.AutoHTTPS)
		})
	}
}

func TestPortalEntryAccessesSeparateEffectiveAndReservedTransports(t *testing.T) {
	entry := NormalizePortalEntry(PortalEntry{Protocol: "http", Http: new(PortalEntryHTTP{Host: "example.com", Http: "DISABLED", HttpsPort: 8443})})
	reserved := []PortalEntryAccess{{Scheme: "https", Port: 8443}}
	require.Equal(t, reserved, entry.ReservedAccesses())
	// An unresolved entry must never masquerade as an effective listener.
	require.Panics(t, func() { entry.EffectiveAccesses() })

	for _, hasCertificate := range []bool{false, true, false} {
		entry.EffectiveHTTP = new(entry.Http.Effective(hasCertificate))
		if hasCertificate {
			require.Equal(t, reserved, entry.EffectiveAccesses())
		} else {
			require.Empty(t, entry.EffectiveAccesses())
		}
		require.Equal(t, reserved, entry.ReservedAccesses())
	}
}
func TestPortalHTTPRejectsExplicitConflicts(t *testing.T) {
	for _, raw := range []PortalEntryHTTP{
		{Https: "ENABLED"}, {Host: "::1", Https: "ENABLED"}, {AutoHTTPS: "ENABLED"},
		{Host: "example.com", Https: "DISABLED", AutoHTTPS: "ENABLED"},
		{Host: "example.com", Http: "DISABLED", AutoHTTPS: "ENABLED"},
		{Http: "DISABLED", Https: "DISABLED"},
		{Host: "example.com", HttpPort: 443}, {HttpPort: -1}, {HttpsPort: 65536},
	} {
		require.Panics(t, func() { NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &raw}) })
	}
	require.NotPanics(t, func() { NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &PortalEntryHTTP{HttpPort: 443}}) })
}
func TestPortalCertificatePresenceIgnoresValidity(t *testing.T) {
	for _, date := range []time.Time{time.Now().Add(-time.Hour), time.Now().Add(time.Hour)} {
		certs := []*PortalCert{{Enabled: true, Domains: []string{"*.example.com"}, ValidFrom: date, ValidTo: date}}
		require.True(t, HasPortalCertificate("app.example.com", certs))
		require.True(t, HasPortalCertificate("*.example.com", certs))
		require.False(t, HasPortalCertificate("a.b.example.com", certs))
		require.False(t, HasPortalCertificate("example.com", certs))
		certs[0].Enabled = false
		require.False(t, HasPortalCertificate("app.example.com", certs))
	}
	require.False(t, HasPortalCertificate("*.example.com", []*PortalCert{{Enabled: true, Domains: []string{"app.example.com"}}}))
}

func TestPortalEntryExplicitSwitchesRequireMatchingCertificate(t *testing.T) {
	for _, raw := range []PortalEntryHTTP{
		{Host: "app.example.com", Https: "ENABLED"},
		{Host: "app.example.com", AutoHTTPS: "ENABLED"},
		{Host: "*.example.com", Https: "ENABLED"},
	} {
		entry := NormalizePortalEntry(PortalEntry{Name: "web", Protocol: "http", Http: &raw})
		for _, certs := range [][]*PortalCert{
			nil,
			{{Enabled: false, Domains: []string{"*.example.com"}}},
			{{Enabled: true, Domains: []string{"other.example.com"}}},
		} {
			require.Panics(t, func() { ValidatePortalEntryCertificates(entry, certs) })
		}
		for _, date := range []time.Time{time.Now().Add(-time.Hour), time.Now().Add(time.Hour)} {
			require.NotPanics(t, func() {
				ValidatePortalEntryCertificates(entry, []*PortalCert{{Enabled: true, Domains: []string{"*.example.com"}, ValidFrom: date, ValidTo: date}})
			})
		}
	}
	for _, raw := range []PortalEntryHTTP{{Host: "app.example.com"}, {Host: "app.example.com", Https: "DISABLED"}} {
		entry := NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &raw})
		require.NotPanics(t, func() { ValidatePortalEntryCertificates(entry, nil) })
	}
}

func TestPortalHTTPModeJSON(t *testing.T) {
	for _, input := range []string{`{}`, `{"http":"AUTO","https":"AUTO","autohttps":"AUTO"}`} {
		raw, err := vcode.UnmarshalJsonS[PortalEntryHTTP](input)
		require.NoError(t, err)
		entry := NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &raw})
		require.Equal(t, PortalSwitchModeAuto, entry.Http.Http)
		require.Equal(t, PortalSwitchModeAuto, entry.Http.Https)
		require.Equal(t, PortalSwitchModeAuto, entry.Http.AutoHTTPS)
		encoded := vcode.MustMarshalJsonS(entry.Http)
		require.JSONEq(t, `{"host":"","http":"AUTO","httpPort":80,"https":"AUTO","httpsPort":443,"autohttps":"AUTO"}`, encoded)
	}
	for _, field := range []string{"http", "https", "autohttps"} {
		for _, value := range []string{`null`, `true`, `false`, `1`, `""`, `"enabled"`, `"ON"`, `"UNSPECIFIED"`} {
			_, err := vcode.UnmarshalJsonS[PortalEntryHTTP](fmt.Sprintf(`{"%s":%s}`, field, value))
			require.Error(t, err, "%s=%s", field, value)
		}
	}
	for _, mode := range []PortalSwitchMode{"enabled", "ON", "UNSPECIFIED"} {
		for _, raw := range []PortalEntryHTTP{{Http: mode}, {Https: mode}, {AutoHTTPS: mode}} {
			require.Panics(t, func() { NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &raw}) })
		}
	}
}
