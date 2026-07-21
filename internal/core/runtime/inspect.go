package runtime

import "fmt"

func Inspect() string {
	info := Application()
	desc := fmt.Sprintf("AppInfo\n")
	desc += fmt.Sprintf("  ├ Name = %s\n", info.Name())
	desc += fmt.Sprintf("  ├ Version = %s\n", info.Version())
	desc += fmt.Sprintf("  ├ GitCommit = %s\n", GitCommit())
	desc += fmt.Sprintf("  ├ BuiltBy = %s\n", BuiltBy())
	desc += fmt.Sprintf("  ├ BuiltTime = %s\n", BuiltTime())
	desc += fmt.Sprintf("  ├ GoVersion = %s\n", GoVersion())
	desc += fmt.Sprintf("  ├ GoCompiler = %s\n", GoCompiler())
	desc += fmt.Sprintf("  └ GoPlatform = %s\n", GoPlatform())
	return desc
}
