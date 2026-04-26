// Package version holds the tool's build version. Set via -ldflags at build time.
// When installed via `go install ...@latest`, the module version is read from
// the embedded build info as a fallback.
package version

import "runtime/debug"

// Version is overwritten by the linker at build time. When not set (e.g.,
// `go install` without -ldflags), it falls back to the module version from
// Go's embedded build info.
var Version = func() string {
	if v := linkerVersion; v != "" {
		return v
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}()

// linkerVersion is set via -ldflags:
//
//	go build -ldflags "-X github.com/shibuiwilliam/archfit/internal/version.linkerVersion=1.0.0"
var linkerVersion string
