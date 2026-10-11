package core

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPortalHTTPDefaultsAndSingleTransport(t *testing.T) {
	entry := NormalizePortalEntry(PortalEntry{Protocol: "http"})
	require.Equal(t, DefaultPortalEntryHTTP(), *entry.Http)
	require.Len(t, entry.Accesses(), 2)
	for _, scheme := range []string{"http", "https"} {
		single := NormalizePortalEntry(PortalEntry{Id: 42, Name: scheme, Enabled: true, ListenIPs: []string{"127.0.0.1"}, Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: scheme == "http", HttpsEnabled: scheme == "https", HttpPort: 8443, HttpsPort: 8443}})
		require.Equal(t, 42, single.Id)
		require.Equal(t, "http", single.Protocol)
		require.Equal(t, scheme, single.Accesses()[0].Scheme)
		require.Equal(t, 8443, single.Accesses()[0].Port)
		require.False(t, single.Http.AutoHTTPS)
		require.Len(t, single.Accesses(), 1)
		require.Equal(t, single, NormalizePortalEntry(single))
	}
}
func TestPortalHTTPInvalidConfigurations(t *testing.T) {
	for _, config := range []PortalEntryHTTP{
		{}, {AutoHTTPS: true},
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
			{Name: "dual", Protocol: "http", Enabled: true, Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: true, HttpsEnabled: true, AutoHTTPS: true}},
			{Name: "single", Enabled: true, Protocol: "http", Http: &PortalEntryHTTP{Host: "demo.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 80}},
		})
	})
	require.NotPanics(t, func() {
		ValidatePortalEntryListeners([]PortalEntry{
			{Name: "dual", Protocol: "http", Enabled: true, Http: &PortalEntryHTTP{Host: "one.local", HttpEnabled: true, HttpsEnabled: true, AutoHTTPS: true}},
			{Name: "single", Enabled: true, Protocol: "http", Http: &PortalEntryHTTP{Host: "two.local", HttpEnabled: true, HttpsEnabled: false, HttpPort: 80}},
		})
	})
}

func TestPortalHTTPPreservesAutoHTTPSWithSingleTransport(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		t.Run(scheme, func(t *testing.T) {
			config := DefaultPortalEntryHTTP()
			config.HttpEnabled = scheme == "http"
			config.HttpsEnabled = scheme == "https"
			entry := NormalizePortalEntry(PortalEntry{Protocol: "http", Http: &config})
			require.True(t, entry.Http.AutoHTTPS)
			require.Len(t, entry.Accesses(), 1)
			require.Equal(t, scheme, entry.Accesses()[0].Scheme)
			entry.Http.HttpEnabled = true
			entry.Http.HttpsEnabled = true
			entry = NormalizePortalEntry(entry)
			require.True(t, entry.Http.AutoHTTPS)
			require.Len(t, entry.Accesses(), 2)
		})
	}
}
