package version

const (
	// Name of the software
	AppName = "MCP-Tester"
	// Author of the software
	Author = "Michael Lechner"
	// Copyright notice
	Copyright = "Copyright © 2026 Michael Lechner"
)

// Version is stamped from the VERSION file at build time via
// `-ldflags -X …/internal/version.Version`.
//
// A var rather than a const, and that is the whole point: it was a const here
// with the number written out, so the released binaries could only ever report
// whatever was last typed into this file. "dev" is what an unstamped build
// gets, which is honest.
var Version = "dev"
