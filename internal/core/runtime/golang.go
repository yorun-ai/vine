package runtime

import (
	"fmt"
	goruntime "runtime"
)

func GoVersion() string {
	return goruntime.Version()
}

func GoCompiler() string {
	return goruntime.Compiler
}

func GoPlatform() string {
	return fmt.Sprintf("%s/%s", goruntime.GOOS, goruntime.GOARCH)
}
