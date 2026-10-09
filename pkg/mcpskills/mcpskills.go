// Package mcpskills serves Agent Skills over MCP (Skills extension,
// io.modelcontextprotocol/skills, on protocol 2026-07-28) from a server built
// on the official go-sdk, which does not support the extension yet. It also
// holds the format checks the mcp-tester uses to verify skills.
//
// Every file of a skill directory becomes a skill://<skill-path>/<file>
// resource; skills/list and skills/get return complete manifests with
// SHA-256 digests; resources/directory/read lists directories.
//
//	caps := &mcp.ServerCapabilities{}
//	mcpskills.Declare(caps, true)
//	s := mcp.NewServer(impl, &mcp.ServerOptions{Capabilities: caps})
//	mcpskills.Serve(s, os.DirFS("skills"), nil)
package mcpskills

import (
	"context"
	"io/fs"
	"mime"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Extension is the identifier of the Skills extension.
const Extension = "io.modelcontextprotocol/skills"

// DirectoryMIMEType marks a directory resource.
const DirectoryMIMEType = "inode/directory"

const (
	codeInvalidParams = -32602
	defaultTTLMs      = 300000
)

// Declare adds the extension (and the resources capability it requires) to
// the server capabilities. directoryRead announces resources/directory/read.
func Declare(caps *mcp.ServerCapabilities, directoryRead bool) {
	caps.AddExtension(Extension, map[string]any{"directoryRead": directoryRead})
	if caps.Resources == nil {
		caps.Resources = &mcp.ResourceCapabilities{}
	}
}

// SkillResource is one file of a skill with its digest and size.
type SkillResource struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

// Skill is a skill entry as returned by skills/list and skills/get.
type Skill struct {
	URI         string          `json:"uri"`
	Frontmatter map[string]any  `json:"frontmatter"`
	Resources   []SkillResource `json:"resources"`
}

// Options configure Serve.
type Options struct {
	// Scheme of the resource URIs; default "skill".
	Scheme string
	// TTLMs is the cache hint on skills/list and skills/get; default 300000.
	TTLMs int
}

type file struct {
	uri  string
	data []byte
	mime string
}

type catalog struct {
	scheme string
	skills []Skill
	byURI  map[string]Skill
	files  map[string]*file
	dirs   map[string][]*mcp.Resource // directory URI -> direct children
	ttlMs  int
}

// Serve publishes every skill (directory with a SKILL.md) in fsys and
// returns the entries. A skill whose frontmatter name differs from its
// directory name, or that breaks the Agent Skills rules, is an error.
func Serve(s *mcp.Server, fsys fs.FS, opts *Options) ([]Skill, error) {
	scheme, ttl := "skill", defaultTTLMs
	if opts != nil && opts.Scheme != "" {
		scheme = opts.Scheme
	}
	if opts != nil && opts.TTLMs > 0 {
		ttl = opts.TTLMs
	}
	c, err := load(fsys, scheme)
	if err != nil {
		return nil, err
	}
	c.ttlMs = ttl

	for _, f := range c.files {
		res := &mcp.Resource{URI: f.uri, Name: path.Base(f.uri), MIMEType: f.mime}
		if sk, ok := c.byURI[f.uri]; ok {
			// SKILL.md: name and description from the frontmatter
			res.Name, _ = sk.Frontmatter["name"].(string)
			res.Description, _ = sk.Frontmatter["description"].(string)
		}
		s.AddResource(res, c.read)
	}
	if err := mcp.AddReceivingCustomMethod(s, "skills/list", c.list); err != nil {
		return nil, err
	}
	if err := mcp.AddReceivingCustomMethod(s, "skills/get", c.get); err != nil {
		return nil, err
	}
	if err := mcp.AddReceivingCustomMethod(s, "resources/directory/read", c.readDir); err != nil {
		return nil, err
	}
	return c.skills, nil
}

func (c *catalog) addChild(dir string, res *mcp.Resource) {
	for _, existing := range c.dirs[dir] {
		if existing.URI == res.URI {
			return
		}
	}
	c.dirs[dir] = append(c.dirs[dir], res)
}

func mimeType(p string, data []byte) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".md", ".markdown":
		return "text/markdown"
	case ".py":
		return "text/x-python"
	case ".sh":
		return "text/x-shellscript"
	}
	if t := mime.TypeByExtension(path.Ext(p)); t != "" {
		return t
	}
	if utf8.Valid(data) {
		return "text/plain"
	}
	return "application/octet-stream"
}

func (c *catalog) read(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	f, ok := c.files[req.Params.URI]
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}
	contents := &mcp.ResourceContents{URI: f.uri, MIMEType: f.mime}
	// The digest covers the raw bytes: text is sent as is, anything else as blob
	if utf8.Valid(f.data) {
		contents.Text = string(f.data)
	} else {
		contents.Blob = f.data
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{contents}}, nil
}
