// Package version holds the tool's build version. Set via -ldflags at build time.
package version

// Version is overwritten by the linker. Default "dev" keeps go run usable.
var Version = "dev"
