package cli

import (
	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/vine/internal/core/mtls"
)

const (
	FlagMTLSCAFile   = "mtls-ca-file"
	FlagMTLSCertFile = "mtls-cert-file"
	FlagMTLSKeyFile  = "mtls-key-file"

	EnvMTLSCAFile   = "VINE_MTLS_CA_FILE"
	EnvMTLSCertFile = "VINE_MTLS_CERT_FILE"
	EnvMTLSKeyFile  = "VINE_MTLS_KEY_FILE"
)

func mtlsFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: FlagMTLSCAFile, Sources: ucli.EnvVars(EnvMTLSCAFile), Usage: "Vine backend mTLS CA certificate file"},
		&ucli.StringFlag{Name: FlagMTLSCertFile, Sources: ucli.EnvVars(EnvMTLSCertFile), Usage: "this component's mTLS certificate file"},
		&ucli.StringFlag{Name: FlagMTLSKeyFile, Sources: ucli.EnvVars(EnvMTLSKeyFile), Usage: "this component's mTLS private key file"},
	}
}

func mtlsFiles(cmd *ucli.Command) mtls.Files {
	return mtls.Files{
		CAFile:   cmd.String(FlagMTLSCAFile),
		CertFile: cmd.String(FlagMTLSCertFile),
		KeyFile:  cmd.String(FlagMTLSKeyFile),
	}
}
