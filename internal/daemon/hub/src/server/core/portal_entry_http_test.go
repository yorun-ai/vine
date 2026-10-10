package core

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPortalHTTPDefaultsAndLegacyConversion(t *testing.T) {
	entry := NormalizePortalEntry(PortalEntry{Protocol: "http"})
	require.Equal(t, DefaultPortalEntryHTTP(), *entry.Http)
	require.Len(t, entry.Accesses(), 2)
	for _, scheme := range []string{"http", "https"} {
		legacy := NormalizePortalEntry(PortalEntry{Id: 42, Name: scheme, Scheme: scheme, Port: 8443, Host: "demo.local", Enabled: true, ListenIPs: []string{"127.0.0.1"}})
		require.Equal(t, 42, legacy.Id)
		require.Equal(t, "http", legacy.Protocol)
		require.Equal(t, scheme, legacy.Scheme)
		require.Equal(t, 8443, legacy.Port)
		require.False(t, legacy.Http.AutoHTTPS)
		require.Len(t, legacy.Accesses(), 1)
		require.Equal(t, legacy, NormalizePortalEntry(legacy))
	}
}
func TestPortalHTTPInvalidConfigurations(t *testing.T) {
	for _, config := range []PortalEntryHTTP{
		{}, {HttpEnabled: true, HttpPort: 80, AutoHTTPS: true}, {HttpsEnabled: true, HttpsPort: 443, AutoHTTPS: true},
		{HttpEnabled: true, HttpsEnabled: true, HttpPort: 8080, HttpsPort: 8080},
		{HttpEnabled: true, HttpPort: -1}, {HttpsEnabled: true, HttpsPort: 65536},
	} {
		require.Panics(t, func() { NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &config}) })
	}
	require.Panics(t, func() { NormalizePortalEntry(PortalEntry{Protocol: "tcp"}) })
}
func TestPortalHTTPRejectsPartiallyOverlappingEntry(t *testing.T) {
	require.Panics(t, func() {
		ValidatePortalEntryListeners([]PortalEntry{
			{Name: "dual", Protocol: "http", Host: "demo.local", Enabled: true},
			{Name: "single", Scheme: "http", Port: 80, Host: "demo.local", Enabled: true},
		})
	})
	require.NotPanics(t, func() {
		ValidatePortalEntryListeners([]PortalEntry{
			{Name: "dual", Protocol: "http", Host: "one.local", Enabled: true},
			{Name: "single", Scheme: "http", Port: 80, Host: "two.local", Enabled: true},
		})
	})
}
