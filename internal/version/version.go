package version

import (
	"runtime/debug"
	"strings"
)

const (
	// AppName is the name of the software
	AppName = "MCP-Tester"
	// Author of the software
	Author = "Michael Lechner"
	// Copyright notice
	Copyright = "Copyright © 2026 Michael Lechner"
)

// Version is stamped from the VERSION file at build time via
// `-ldflags -X …/internal/version.Version`.
//
// If unstamped ("dev" or empty), it falls back to debug.ReadBuildInfo()
// to retrieve the module version tag when installed via `go install`.
var Version = "dev"

func init() {
	if bi, ok := debug.ReadBuildInfo(); ok {
		Version = resolveVersion(Version, bi)
	}
}

// resolveVersion determines the version using ldflags or Go module build info.
func resolveVersion(current string, bi *debug.BuildInfo) string {
	if current != "dev" && current != "" {
		return current
	}
	if bi != nil && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return "dev"
}
