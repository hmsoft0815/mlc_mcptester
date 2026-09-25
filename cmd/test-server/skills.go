package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed skills
var skillFiles embed.FS

// registerSkills publishes the embedded sample skills (Skills extension).
func registerSkills(s *mcp.Server) {
	sub, err := fs.Sub(skillFiles, "skills")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := mcpskills.Serve(s, sub, nil); err != nil {
		log.Fatalf("serving skills: %v", err)
	}
}
