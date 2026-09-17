package linked

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/appcli"
	linkflag "go.yorun.ai/vine/internal/daemon/link/src/server/flag"
)

// Option configures the Hub connection and ingress listener for linked mode.
type Option struct {
	// LinkHubEndpoint is the API endpoint of the external Hub.
	LinkHubEndpoint string
	// LinkIngressListen is the address on which the in-process Link accepts application traffic.
	LinkIngressListen string
	// LinkMTLSCAFile is the CA certificate used to authenticate Vine backend components.
	LinkMTLSCAFile string
	// LinkMTLSCertFile is the in-process Link's X.509-SVID certificate.
	LinkMTLSCertFile string
	// LinkMTLSKeyFile is the private key for the in-process Link's certificate.
	LinkMTLSKeyFile string

	// IgnoredFlags lists the flags the binary accepts but discards, named with the
	// Flag constants of this package. The named flags and their environment
	// variables stop reaching the runtime, so an embedding program can own that
	// parameter; setting the matching Option field still applies it.
	IgnoredFlags []string

	// RenamedFlags maps a declared flag name, such as FlagMTLSKeyFile, to the name
	// the binary registers it under. The declared flag and its environment variable
	// are dropped: the new name carries the environment variable derived from it. A
	// renamed flag cannot also be ignored.
	RenamedFlags map[string]string
}

func (o Option) isZero() bool {
	return o.LinkHubEndpoint == "" &&
		o.LinkIngressListen == "" &&
		o.LinkMTLSCAFile == "" &&
		o.LinkMTLSCertFile == "" &&
		o.LinkMTLSKeyFile == "" &&
		len(o.IgnoredFlags) == 0 &&
		len(o.RenamedFlags) == 0
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

// flags lists the Link parameters the business binary accepts, as the option
// presents them: a name it ignores keeps its command line and environment source
// but stops reaching the runtime, and a name it renames answers to the new name
// alone.
func flags(flag *linkflag.Flag, option Option) []ucli.Flag {
	names := appcli.NewFlagNames(option.IgnoredFlags, option.RenamedFlags)

	list := []ucli.Flag{
		names.String(FlagHubEndpoint, EnvHubEndpoint, &flag.HubEndpoint, "Hub API endpoint"),
		names.String(FlagIngressListen, EnvIngressListen, &flag.IngressListen, "in-process Link ingress listen address"),
		names.String(FlagMTLSCAFile, EnvMTLSCAFile, &flag.MTLS.CAFile, "Vine backend mTLS CA certificate file for the in-process Link"),
		names.String(FlagMTLSCertFile, EnvMTLSCertFile, &flag.MTLS.CertFile, "the in-process Link's mTLS certificate file"),
		names.String(FlagMTLSKeyFile, EnvMTLSKeyFile, &flag.MTLS.KeyFile, "the in-process Link's mTLS private key file"),
	}

	names.Validate()
	return list
}

// applyOption copies the declared option over the parsed flags: a value the
// program sets wins over the command line and the environment.
func applyOption(flag *linkflag.Flag, option Option) {
	if option.LinkHubEndpoint != "" {
		flag.HubEndpoint = option.LinkHubEndpoint
	}
	if option.LinkIngressListen != "" {
		flag.IngressListen = option.LinkIngressListen
	}
	if option.LinkMTLSCAFile != "" {
		flag.MTLS.CAFile = option.LinkMTLSCAFile
	}
	if option.LinkMTLSCertFile != "" {
		flag.MTLS.CertFile = option.LinkMTLSCertFile
	}
	if option.LinkMTLSKeyFile != "" {
		flag.MTLS.KeyFile = option.LinkMTLSKeyFile
	}
}
