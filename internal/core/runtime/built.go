package runtime

import (
	"go.yorun.ai/vine/buildinfo"
)

func GitCommit() string {
	value, _ := buildinfo.GitCommit()
	return value
}

func BuiltBy() string {
	value, _ := buildinfo.BuiltBy()
	return value
}

func BuiltTime() string {
	value, _ := buildinfo.BuiltTime()
	return value
}
