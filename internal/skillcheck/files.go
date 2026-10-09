package skillcheck

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func checkManifest(skillURI, root string, manifest []mcpskills.SkillResource, total *int64) []string {
	var problems []string
	seen := map[string]bool{}
	hasSkillMD := false
	for _, r := range manifest {
		switch {
		case seen[r.URI]:
			problems = append(problems, fmt.Sprintf("%s is listed twice", r.URI))
		case !strings.HasPrefix(r.URI, root+"/"):
			problems = append(problems, fmt.Sprintf("%s is outside the skill directory", r.URI))
		case !mcpskills.ValidDigest(r.Digest):
			problems = append(problems, fmt.Sprintf("%s: digest %q must be sha256:<64 lowercase hex>", r.URI, r.Digest))
		case r.Size < 0:
			problems = append(problems, fmt.Sprintf("%s: negative size", r.URI))
		}
		seen[r.URI] = true
		hasSkillMD = hasSkillMD || r.URI == skillURI
		*total += r.Size
	}
	if !hasSkillMD {
		problems = append(problems, "the manifest does not list SKILL.md itself")
	}
	return problems
}

// verifyFiles reads every listed file and compares it with the manifest.
func (c *Checker) verifyFiles(ctx context.Context, label string, e entry, manifest []mcpskills.SkillResource, add func(string, Status, string, ...any)) {
	var problems []string
	for _, r := range manifest {
		data, err := c.readFile(ctx, r.URI)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", r.URI, err))
			continue
		}
		if int64(len(data)) != r.Size {
			problems = append(problems, fmt.Sprintf("%s: %d bytes, manifest says %d", r.URI, len(data), r.Size))
		} else if mcpskills.Digest(data) != r.Digest {
			problems = append(problems, fmt.Sprintf("%s: digest does not match the manifest", r.URI))
		}
		if r.URI == e.URI {
			fm, err := mcpskills.ParseFrontmatter(data)
			switch {
			case err != nil:
				problems = append(problems, err.Error())
			case !sameJSON(fm, e.Frontmatter):
				problems = append(problems, "the SKILL.md frontmatter differs from the entry's frontmatter")
			}
		}
	}
	if len(problems) > 0 {
		add(label+" verify", Fail, "%s", strings.Join(problems, "; "))
	} else {
		add(label+" verify", Pass, "%d file(s): sizes, digests and frontmatter match", len(manifest))
	}
}

// readFile returns the raw bytes of a resource.
func (c *Checker) readFile(ctx context.Context, uri string) ([]byte, error) {
	res, err := c.Session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return nil, err
	}
	if len(res.Contents) != 1 {
		return nil, fmt.Errorf("%d contents, want 1", len(res.Contents))
	}
	if res.Contents[0].Blob != nil {
		return res.Contents[0].Blob, nil
	}
	return []byte(res.Contents[0].Text), nil
}

// checkDirectory compares the direct children of the skill root with the manifest.
func (c *Checker) checkDirectory(ctx context.Context, label, root string, manifest []mcpskills.SkillResource, add func(string, Status, string, ...any)) {
	res, err := c.call(ctx, "resources/directory/read", map[string]any{"uri": root})
	if err != nil {
		add(label+" directory read", Fail, "%v", err)
		return
	}
	var body struct {
		Resources []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
		} `json:"resources"`
	}
	if err := remarshal(res, &body); err != nil {
		add(label+" directory read", Fail, "invalid result: %v", err)
		return
	}
	listed := map[string]bool{}
	for _, r := range manifest {
		listed[r.URI] = true
	}
	var unlisted []string
	for _, child := range body.Resources {
		if child.MIMEType != mcpskills.DirectoryMIMEType && manifest != nil && !listed[child.URI] {
			unlisted = append(unlisted, child.URI)
		}
	}
	if len(unlisted) > 0 {
		add(label+" directory read", Warn, "files not in the manifest (stale entry or changed skill): %s", strings.Join(unlisted, ", "))
	} else {
		add(label+" directory read", Pass, "%d child(ren), consistent with the manifest", len(body.Resources))
	}
}
