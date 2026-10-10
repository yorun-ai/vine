package entry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/require"
	hubapiwatch "go.yorun.ai/vine/internal/daemon/hub/api/watch"
	"go.yorun.ai/vine/internal/daemon/hub/api/watched"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/vault"
	"go.yorun.ai/vine/internal/utilfortest/watchtest"
	"go.yorun.ai/vine/util/vcode"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAutoHTTPSPreservesRequestURL(t *testing.T) {
	for _, test := range []struct {
		host     string
		port     int
		expected string
	}{
		{"demo.local:8080", 443, "https://demo.local/a%2Fb?q=x%2Fy"},
		{"demo.local:8080", 8443, "https://demo.local:8443/a%2Fb?q=x%2Fy"},
		{"[::1]:8080", 443, "https://[::1]/a%2Fb?q=x%2Fy"},
		{"[::1]:8080", 8443, "https://[::1]:8443/a%2Fb?q=x%2Fy"},
	} {
		request := httptest.NewRequest(http.MethodPost, "http://ignored/a%2Fb?q=x%2Fy", nil)
		request.Host = test.host
		request.Header.Set("X-Forwarded-Host", "attacker.local")
		response := httptest.NewRecorder()
		rule := _Rule{autoHTTPSPort: test.port}
		rule.Serve(&spec.Context{Request: request, ResponseWriter: response})
		require.Equal(t, 308, response.Code)
		require.Equal(t, test.expected, response.Header().Get("Location"))
	}
}
func TestHTTPEntryWatchCreatesRedirectWithoutRulesAndUpdatesTransports(t *testing.T) {
	certificateVault := &vault.Vault{Context: context.Background(), Watch: watchtest.New(t, map[string]string{watched.FormatPortalCertKey("test"): vcode.MustMarshalJsonS(testHTTPPortalCertificate(t))})}
	certificateVault.DIInit()
	manager := &Manager{Vault: certificateVault, entryRulesByName: map[string]watched.PortalRule{}, entryConfigsByName: map[string]watched.PortalEntry{}, entriesByKey: map[_Key]*_Entry{}}
	config := watched.PortalEntry{Name: "web", Protocol: "http", Host: "", ListenIPs: []string{"127.0.0.1"}, Http: watched.PortalEntryHTTP{HttpEnabled: true, HttpPort: 8080, HttpsEnabled: true, HttpsPort: 8443, AutoHTTPS: true}}
	event := hubapiwatch.Event{Key: watched.FormatPortalEntryKey("web"), Value: vcode.MustMarshalJsonS(config)}
	manager.handlePortalEntryEvent(event)
	require.Len(t, manager.entriesByKey, 2)
	ingress := manager.entriesByKey[_Key{listenIP: "127.0.0.1", scheme: spec.SchemeHTTP, port: 8080}]
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "http://example.com/a", nil)
	ingress.ServeHTTP(response, request)
	require.Equal(t, 308, response.Code)
	require.Equal(t, "https://example.com:8443/a", response.Header().Get("Location"))
	require.Empty(t, manager.entriesByKey[_Key{listenIP: "127.0.0.1", scheme: spec.SchemeHTTPS, port: 8443}].rules)
	config.Http.AutoHTTPS = false
	event.Value = vcode.MustMarshalJsonS(config)
	manager.handlePortalEntryEvent(event)
	response = httptest.NewRecorder()
	ingress.ServeHTTP(response, request)
	require.Equal(t, 404, response.Code)
	manager.entryRulesByName["rule"] = watched.PortalRule{Name: "rule", EntryName: "web", RouteType: routeTypePermanentRedirect, RouteRedirectionPattern: "https://target.local", ResolvedMatchPathPrefix: "/a"}
	config.Host = "example.com"
	event.Value = vcode.MustMarshalJsonS(config)
	manager.handlePortalEntryEvent(event)
	require.Len(t, ingress.rules, 1)
	require.Len(t, manager.entriesByKey[_Key{listenIP: "127.0.0.1", scheme: spec.SchemeHTTPS, port: 8443}].rules, 1)
	config.Http.HttpsEnabled = false
	event.Value = vcode.MustMarshalJsonS(config)
	manager.handlePortalEntryEvent(event)
	require.Len(t, manager.entriesByKey, 1)
	event.Kind = hubapiwatch.EventKindDelete
	manager.handlePortalEntryEvent(event)
	require.Empty(t, manager.entriesByKey)
}

func TestHTTPProtocolRealRedirectAndTLSListeners(t *testing.T) {
	cert := testHTTPPortalCertificate(t)
	certificate, err := tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
	require.NoError(t, err)
	parsed := certificate.Leaf
	certificateVault := &vault.Vault{Context: context.Background(), Watch: watchtest.New(t, map[string]string{watched.FormatPortalCertKey("test"): vcode.MustMarshalJsonS(cert)})}
	certificateVault.DIInit()
	httpReservation, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	httpsReservation, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	httpPort := httpReservation.Addr().(*net.TCPAddr).Port
	httpsPort := httpsReservation.Addr().(*net.TCPAddr).Port
	config := watched.PortalEntry{Name: "web", Protocol: "http", Host: "example.com", ListenIPs: []string{"127.0.0.1"}, Http: watched.PortalEntryHTTP{HttpEnabled: true, HttpPort: httpPort, HttpsEnabled: true, HttpsPort: httpsPort, AutoHTTPS: true}}
	clientWatch := watchtest.New(t, map[string]string{watched.FormatPortalEntryKey("web"): vcode.MustMarshalJsonS(config)})
	manager := &Manager{Context: context.Background(), Watch: clientWatch, Vault: certificateVault}
	manager.DIInit()
	require.NoError(t, httpReservation.Close())
	require.NoError(t, httpsReservation.Close())
	manager.AfterAppStart()
	t.Cleanup(manager.AfterAppStop)
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: "example.com"}, DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		return (&net.Dialer{}).DialContext(ctx, "tcp4", net.JoinHostPort("127.0.0.1", port))
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://example.com:%d/a%%2Fb?q=x%%2Fy", httpPort), nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.NotNil(t, response.TLS)
	require.Equal(t, http.MethodPost, response.Request.Method)
	require.Equal(t, fmt.Sprintf("https://example.com:%d/a%%2Fb?q=x%%2Fy", httpsPort), response.Request.URL.String())
}

func testHTTPPortalCertificate(t *testing.T) watched.PortalCert {
	t.Helper()
	server := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := server.TLS.Certificates[0]
	server.Close()
	parsed, err := x509.ParseCertificate(certificate.Certificate[0])
	require.NoError(t, err)
	private, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	require.NoError(t, err)
	return watched.PortalCert{Name: "test", Certificate: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]})), PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})), ValidFrom: parsed.NotBefore, ValidTo: parsed.NotAfter}
}

func TestAutoHTTPSFallsBackToHTTPWithoutValidCertificate(t *testing.T) {
	certificateWatch := watchtest.New(t, nil)
	certificateVault := &vault.Vault{Context: context.Background(), Watch: certificateWatch}
	certificateVault.DIInit()
	config := watched.PortalEntry{Name: "web", Protocol: "http", Http: watched.PortalEntryHTTP{HttpEnabled: true, HttpPort: 8080, HttpsEnabled: true, HttpsPort: 8443, AutoHTTPS: true}}
	manager := &Manager{Vault: certificateVault, entryConfigsByName: testEntryConfigs(config), entryRulesByName: map[string]watched.PortalRule{"rule": testRedirectRule("rule", "web")}, entriesByKey: map[_Key]*_Entry{}}
	require.NoError(t, manager.reconcileEntriesLocked())
	ingress := manager.entriesByKey[_Key{scheme: spec.SchemeHTTP, port: 8080}]
	location := func(host string) string {
		response := httptest.NewRecorder()
		ingress.ServeHTTP(response, httptest.NewRequest("GET", "http://"+host+"/a", nil))
		return response.Header().Get("Location")
	}
	require.Equal(t, "https://example.com", location("example.com"), "the configured HTTP rule still runs")
	cert := testHTTPPortalCertificate(t)
	certificateWatch.Publish(hubapiwatch.Event{Key: watched.FormatPortalCertKey(cert.Name), Value: vcode.MustMarshalJsonS(cert)})
	require.Eventually(t, func() bool { return location("example.com") == "https://example.com:8443/a" }, time.Second, time.Millisecond)
	require.Equal(t, "https://example.com", location("other.local"), "a certificate must match the actual request host")
	certificateWatch.Publish(hubapiwatch.Event{Kind: hubapiwatch.EventKindDelete, Key: watched.FormatPortalCertKey(cert.Name)})
	require.Eventually(t, func() bool { return location("example.com") == "https://example.com" }, time.Second, time.Millisecond)
}
