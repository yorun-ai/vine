package flag

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yorun.ai/vine/internal/core/mtls"
)

func TestNormalizeRequiresHTTPSHubEndpointWithMTLS(t *testing.T) {
	flags := Flag{
		HubEndpoint: "http://demo.local:7071",
		MTLS:        mtls.Files{CAFile: "ca.pem", CertFile: "cert.pem", KeyFile: "key.pem"},
	}
	require.PanicsWithError(t, "hub-endpoint must use https when mTLS is enabled", func() {
		flags.Normalize()
	})
}

func TestNormalizeRequiresHubEndpoint(t *testing.T) {
	flags := Flag{}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	flags.Normalize()
}

func TestNormalizeAcceptsHubEndpoint(t *testing.T) {
	flags := Flag{HubEndpoint: "http://demo.local:7071"}

	flags.Normalize()

	if flags.HubEndpointURL == nil {
		t.Fatal("expected parsed hub endpoint url")
	}
	if got := flags.HubEndpointURL.Hostname(); got != "demo.local" {
		t.Fatalf("unexpected hub endpoint host: %s", got)
	}
}

func TestNormalizeUsesInprocHubEndpointWhenConfigured(t *testing.T) {
	flags := Flag{HubInprocMode: true}

	flags.Normalize()

	if got := flags.HubEndpoint; got != "rpc+inproc://vine/hub" {
		t.Fatalf("unexpected hub endpoint: %s", got)
	}
	if flags.HubEndpointURL == nil {
		t.Fatal("expected parsed inproc hub endpoint url")
	}
}
