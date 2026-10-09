package mcpskills

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// load reads every skill (directory with a SKILL.md) in fsys into a catalog.
func load(fsys fs.FS, scheme string) (*catalog, error) {
	c := &catalog{
		scheme: scheme,
		byURI:  map[string]Skill{}, files: map[string]*file{}, dirs: map[string][]*mcp.Resource{},
	}
	roots, err := skillRoots(fsys)
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		sk, err := c.loadSkill(fsys, root)
		if err != nil {
			return nil, err
		}
		c.skills = append(c.skills, sk)
		c.byURI[sk.URI] = sk
	}
	return c, nil
}

func (c *catalog) uriOf(p string) string { return c.scheme + "://" + p }

// skillRoots returns the directories holding a SKILL.md, sorted.
func skillRoots(fsys fs.FS) ([]string, error) {
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
	sort.Strings(roots)
	return roots, err
}

// loadSkill checks the skill at root and records all of its files.
func (c *catalog) loadSkill(fsys fs.FS, root string) (Skill, error) {
	fm, err := readFrontmatter(fsys, root)
	if err != nil {
		return Skill{}, err
	}
	// Every file below the root, nested skills included (they are supporting files)
	sk := Skill{URI: c.uriOf(root + "/SKILL.md"), Frontmatter: fm}
	var total int64
	err = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			c.addDir(p, root, d.Name())
			return nil
		}
		res, err := c.addFile(fsys, p, d.Name())
		if err != nil {
			return err
		}
		sk.Resources = append(sk.Resources, res)
		total += res.Size
		return nil
	})
	if err != nil {
		return Skill{}, err
	}
	if len(sk.Resources) > MaxResources || total > MaxTotalSize {
		return Skill{}, fmt.Errorf("%s: %d files / %d bytes exceed the limits (%d files, %d bytes)", root, len(sk.Resources), total, MaxResources, MaxTotalSize)
	}
	return sk, nil
}

// readFrontmatter reads the SKILL.md at root and checks its frontmatter
// against the Agent Skills rules and the directory name.
func readFrontmatter(fsys fs.FS, root string) (map[string]any, error) {
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
	return fm, nil
}

// addDir registers the directory p and, below the skill root, links it into
// its parent.
func (c *catalog) addDir(p, root, name string) {
	uri := c.uriOf(p)
	if _, ok := c.dirs[uri]; !ok {
		c.dirs[uri] = []*mcp.Resource{}
	}
	if p != root {
		c.addChild(c.uriOf(path.Dir(p)), &mcp.Resource{URI: uri, Name: name, MIMEType: DirectoryMIMEType})
	}
}

// addFile serves the file p once (nested skills share files with their
// parent) and returns its manifest entry.
func (c *catalog) addFile(fsys fs.FS, p, name string) (SkillResource, error) {
	data, err := fs.ReadFile(fsys, p)
	if err != nil {
		return SkillResource{}, err
	}
	uri := c.uriOf(p)
	if _, ok := c.files[uri]; !ok {
		f := &file{uri: uri, data: data, mime: mimeType(p, data)}
		c.files[uri] = f
		c.addChild(c.uriOf(path.Dir(p)), &mcp.Resource{URI: uri, Name: name, MIMEType: f.mime})
	}
	return SkillResource{URI: uri, Digest: Digest(data), Size: int64(len(data))}, nil
}
