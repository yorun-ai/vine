package linked

import (
	ucli "github.com/urfave/cli/v3"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

// Option configures the Hub connection and ingress listener for linked mode.
type Option struct {
	// HubEndpoint is the API endpoint of the external Hub.
	HubEndpoint string
	// IngressListen is the address on which the in-process Link accepts application traffic.
	IngressListen string
	// MTLSCAFile is the CA certificate used to authenticate Vine backend components.
	MTLSCAFile string
	// MTLSCertFile is the in-process Link's X.509-SVID certificate.
	MTLSCertFile string
	// MTLSKeyFile is the private key for the in-process Link's certificate.
	MTLSKeyFile string
}

func (o Option) isZero() bool {
	return o.HubEndpoint == "" &&
		o.IngressListen == "" &&
		o.MTLSCAFile == "" &&
		o.MTLSCertFile == "" &&
		o.MTLSKeyFile == ""
}

const (
	// The business binary carries no command that could scope the Link parameters
	// it accepts, so its flags and environment variables name the link explicitly.
	// The names are exported for callers that embed the runtime and compose the
	// same command line.
	FlagHubEndpoint   = "link-hub-endpoint"
	FlagIngressListen = "link-ingress-listen"
	FlagMTLSCAFile    = "link-mtls-ca-file"
	FlagMTLSCertFile  = "link-mtls-cert-file"
	FlagMTLSKeyFile   = "link-mtls-key-file"

	EnvHubEndpoint   = "VINE_LINK_HUB_ENDPOINT"
	EnvIngressListen = "VINE_LINK_INGRESS_LISTEN"
	EnvMTLSCAFile    = "VINE_LINK_MTLS_CA_FILE"
	EnvMTLSCertFile  = "VINE_LINK_MTLS_CERT_FILE"
	EnvMTLSKeyFile   = "VINE_LINK_MTLS_KEY_FILE"
)

// flags lists the Link parameters the business binary accepts.
func flags(flag *linkflag.Flag) []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{
			Name:        FlagHubEndpoint,
			Sources:     ucli.EnvVars(EnvHubEndpoint),
			Usage:       "Hub API endpoint",
			Destination: &flag.HubEndpoint,
		},
		&ucli.StringFlag{
			Name:        FlagIngressListen,
			Sources:     ucli.EnvVars(EnvIngressListen),
			Usage:       "in-process Link ingress listen address",
			Destination: &flag.IngressListen,
		},
		&ucli.StringFlag{
			Name:        FlagMTLSCAFile,
			Sources:     ucli.EnvVars(EnvMTLSCAFile),
			Usage:       "Vine backend mTLS CA certificate file for the in-process Link",
			Destination: &flag.MTLS.CAFile,
		},
		&ucli.StringFlag{
			Name:        FlagMTLSCertFile,
			Sources:     ucli.EnvVars(EnvMTLSCertFile),
			Usage:       "the in-process Link's mTLS certificate file",
			Destination: &flag.MTLS.CertFile,
		},
		&ucli.StringFlag{
			Name:        FlagMTLSKeyFile,
			Sources:     ucli.EnvVars(EnvMTLSKeyFile),
			Usage:       "the in-process Link's mTLS private key file",
			Destination: &flag.MTLS.KeyFile,
		},
	}
}

// applyOption copies the declared option over the parsed flags: a value the
// program sets wins over the command line and the environment.
func applyOption(flag *linkflag.Flag, option Option) {
	if option.HubEndpoint != "" {
		flag.HubEndpoint = option.HubEndpoint
	}
	if option.IngressListen != "" {
		flag.IngressListen = option.IngressListen
	}
	if option.MTLSCAFile != "" {
		flag.MTLS.CAFile = option.MTLSCAFile
	}
	if option.MTLSCertFile != "" {
		flag.MTLS.CertFile = option.MTLSCertFile
	}
	if option.MTLSKeyFile != "" {
		flag.MTLS.KeyFile = option.MTLSKeyFile
	}
}
