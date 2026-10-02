package cli

import (
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/core/mtls"
)

const (
	FlagMTLSCAFile   = "mtls-ca-file"
	FlagMTLSCertFile = "mtls-cert-file"
	FlagMTLSKeyFile  = "mtls-key-file"

	EnvMTLSCAFileSuffix   = "MTLS_CA_FILE"
	EnvMTLSCertFileSuffix = "MTLS_CERT_FILE"
	EnvMTLSKeyFileSuffix  = "MTLS_KEY_FILE"
)

func mtlsFlags(component string) []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: FlagMTLSCAFile, Sources: ucli.EnvVars(componentEnvName(component, EnvMTLSCAFileSuffix)), Usage: "Vine backend mTLS CA certificate file"},
		&ucli.StringFlag{Name: FlagMTLSCertFile, Sources: ucli.EnvVars(componentEnvName(component, EnvMTLSCertFileSuffix)), Usage: "this component's mTLS certificate file"},
		&ucli.StringFlag{Name: FlagMTLSKeyFile, Sources: ucli.EnvVars(componentEnvName(component, EnvMTLSKeyFileSuffix)), Usage: "this component's mTLS private key file"},
	}
}

func mtlsFiles(cmd *ucli.Command) mtls.Files {
	return mtls.Files{
		CAFile:   cmd.String(FlagMTLSCAFile),
		CertFile: cmd.String(FlagMTLSCertFile),
		KeyFile:  cmd.String(FlagMTLSKeyFile),
	}
}

// componentEnvName builds a service environment variable from its component
// and an upper-case configuration suffix.
func componentEnvName(component string, suffix string) string {
	return "VINE_" + strings.ToUpper(component) + "_" + suffix
}
