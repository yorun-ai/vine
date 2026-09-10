package main

import (
	// Embed the IANA timezone database so location-aware features such as cron
	// CRON_TZ expressions and database timestamp scanning work in minimal
	// container images that do not ship a system tzdata package.
	_ "time/tzdata"

	"go.yorun.ai/vine/internal/cli"
)

func main() {
	cli.Main()
}
