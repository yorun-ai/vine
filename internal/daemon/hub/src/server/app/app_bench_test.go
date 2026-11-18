package app

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	internalapp "go.yorun.ai/vine/internal/app"
	"go.yorun.ai/vine/internal/core/meta"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/daemon/hub/src/server/flag"
)

func BenchmarkHubAdminRequest(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(b, err)
	address := listener.Addr().String()
	require.NoError(b, listener.Close())
	hub := internalapp.NewInternalInproc[*HubApp](internalapp.With(new(flag.Flag{
		SeedHubData: "{}",
		AdminListen: address,
	})))
	hub.Start()
	b.Cleanup(hub.StopGracefully)

	client := &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}
	b.Cleanup(client.CloseIdleConnections)
	headers := http.Header{}
	headers.Set(rpchttp.HeaderContentType, rpchttp.ContentTypeJson)
	headers.Set(rpchttp.HeaderAccept, rpchttp.ContentTypeJson)
	rpchttp.EncodeClientToHeader(headers, meta.MustNewAppWithRandomId("vine.hub.dashboard", "0.0.1"))
	rpchttp.EncodeTraceToHeader(headers, meta.InitialTrace())
	for _, method := range []string{"AdminApiService/readOnly", "AppConfigApiService/list"} {
		b.Run(method, func(b *testing.B) {
			url := "http://" + address + "/api/invoke/vine.hub.admin." + method
			call := func() {
				request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(`{"arguments":null}`))
				if err != nil {
					b.Fatal(err)
				}
				request.Header = headers.Clone()
				response, err := client.Do(request)
				if err != nil {
					b.Fatal(err)
				}
				defer response.Body.Close()
				if _, err := io.Copy(io.Discard, response.Body); err != nil {
					b.Fatal(err)
				}
				if status := response.Header.Get(rpchttp.HeaderRpcStatus); status != "OK" {
					b.Fatalf("HTTP %d, Rpc status %q", response.StatusCode, status)
				}
			}
			call() // Warm the connection and application singletons before timing.
			b.ReportAllocs()
			for b.Loop() {
				call()
			}
		})
	}
}
