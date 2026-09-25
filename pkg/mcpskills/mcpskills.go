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
	"fmt"
	"io/fs"
	"mime"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
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

func load(fsys fs.FS, scheme string) (*catalog, error) {
	c := &catalog{byURI: map[string]Skill{}, files: map[string]*file{}, dirs: map[string][]*mcp.Resource{}}
	uriOf := func(p string) string { return scheme + "://" + p }

	var roots []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			roots = append(roots, path.Dir(p))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(roots)

	for _, root := range roots {
		if root == "." {
			return nil, fmt.Errorf("SKILL.md at the top of the skills directory: each skill needs its own directory")
		}
		md, err := fs.ReadFile(fsys, path.Join(root, "SKILL.md"))
		if err != nil {
			return nil, err
		}
		fm, err := ParseFrontmatter(md)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", root, err)
		}
		if problems := ValidateFrontmatter(fm); len(problems) > 0 {
			return nil, fmt.Errorf("%s: %s", root, strings.Join(problems, "; "))
		}
		if name := fm["name"].(string); name != path.Base(root) {
			return nil, fmt.Errorf("%s: frontmatter name %q must equal the directory name", root, name)
		}

		// Every file below the root, nested skills included (they are supporting files)
		sk := Skill{URI: uriOf(root + "/SKILL.md"), Frontmatter: fm}
		var total int64
		err = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			uri := uriOf(p)
			if d.IsDir() {
				if _, ok := c.dirs[uri]; !ok {
					c.dirs[uri] = []*mcp.Resource{}
				}
				if p != root {
					c.addChild(uriOf(path.Dir(p)), &mcp.Resource{URI: uri, Name: d.Name(), MIMEType: DirectoryMIMEType})
				}
				return nil
			}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			if _, ok := c.files[uri]; !ok {
				f := &file{uri: uri, data: data, mime: mimeType(p, data)}
				c.files[uri] = f
				c.addChild(uriOf(path.Dir(p)), &mcp.Resource{URI: uri, Name: d.Name(), MIMEType: f.mime})
			}
			sk.Resources = append(sk.Resources, SkillResource{URI: uri, Digest: Digest(data), Size: int64(len(data))})
			total += int64(len(data))
			return nil
		})
		if err != nil {
			return nil, err
		}
		if len(sk.Resources) > MaxResources || total > MaxTotalSize {
			return nil, fmt.Errorf("%s: %d files / %d bytes exceed the limits (%d files, %d bytes)", root, len(sk.Resources), total, MaxResources, MaxTotalSize)
		}
		c.skills = append(c.skills, sk)
		c.byURI[sk.URI] = sk
	}
	return c, nil
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

// wire types

// ListParams are the params of skills/list.
type ListParams struct {
	mcp.ParamsBase
	Cursor string `json:"cursor,omitempty"`
}

// ListResult is the result of skills/list.
type ListResult struct {
	mcp.ResultBase
	ResultType string  `json:"resultType"`
	Skills     []Skill `json:"skills"`
	NextCursor string  `json:"nextCursor,omitempty"`
	TTLMs      int     `json:"ttlMs"`
	CacheScope string  `json:"cacheScope"`
}

// URIParams are the params of skills/get and resources/directory/read.
type URIParams struct {
	mcp.ParamsBase
	URI    string `json:"uri"`
	Cursor string `json:"cursor,omitempty"`
}

// GetResult is the result of skills/get.
type GetResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
	Skill      Skill  `json:"skill"`
	TTLMs      int    `json:"ttlMs"`
	CacheScope string `json:"cacheScope"`
}

// DirectoryResult is the result of resources/directory/read.
type DirectoryResult struct {
	mcp.ResultBase
	ResultType string          `json:"resultType"`
	Resources  []*mcp.Resource `json:"resources"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

func (c *catalog) list(ctx context.Context, _ *mcp.ServerSession, p *ListParams) (*ListResult, error) {
	skills := c.skills
	if skills == nil {
		skills = []Skill{}
	}
	return &ListResult{ResultType: "complete", Skills: skills, TTLMs: c.ttlMs, CacheScope: "public"}, nil
}

func (c *catalog) get(ctx context.Context, _ *mcp.ServerSession, p *URIParams) (*GetResult, error) {
	sk, ok := c.byURI[p.URI]
	if !ok {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: fmt.Sprintf("not a skill served here: %q", p.URI)}
	}
	return &GetResult{ResultType: "complete", Skill: sk, TTLMs: c.ttlMs, CacheScope: "public"}, nil
}

func (c *catalog) readDir(ctx context.Context, _ *mcp.ServerSession, p *URIParams) (*DirectoryResult, error) {
	children, ok := c.dirs[strings.TrimSuffix(p.URI, "/")]
	if !ok {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: fmt.Sprintf("not a directory resource: %q", p.URI)}
	}
	sorted := append([]*mcp.Resource{}, children...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].URI < sorted[j].URI })
	return &DirectoryResult{ResultType: "complete", Resources: sorted}, nil
}
