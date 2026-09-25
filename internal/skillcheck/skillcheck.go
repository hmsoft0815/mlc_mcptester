// Package skillcheck lists the skills a server publishes (Skills extension,
// io.modelcontextprotocol/skills) and checks them against the extension and
// the Agent Skills format: manifests, frontmatter, digests, error codes.
package skillcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Status of one check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL" // a MUST is violated
	Warn Status = "WARN" // a SHOULD is violated or trust is limited
	Skip Status = "SKIP"
)

// Result is the outcome of one check.
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// SkillInfo summarizes one skill for display, e.g. before enabling a server.
type SkillInfo struct {
	URI          string `json:"uri"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	License      string `json:"license,omitempty"`
	AllowedTools string `json:"allowedTools,omitempty"`
	Files        int    `json:"files"`   // -1 for "dynamic"
	Bytes        int64  `json:"bytes"`   // sum of the manifest sizes
	Dynamic      bool   `json:"dynamic"` // no manifest, no integrity
}

// Report is the outcome of a check run.
type Report struct {
	Declared      bool        `json:"declared"`
	DirectoryRead bool        `json:"directoryRead"`
	Skills        []SkillInfo `json:"skills"`
	Results       []Result    `json:"results"`
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == Fail {
			return true
		}
	}
	return false
}

// entry is a skill entry as it arrives on the wire.
type entry struct {
	URI         string          `json:"uri"`
	Frontmatter map[string]any  `json:"frontmatter"`
	Resources   json.RawMessage `json:"resources"`
}

// Checker runs the checks over a session.
type Checker struct {
	Session *mcp.ClientSession
	// Verify reads every file and compares size, digest and frontmatter.
	Verify bool
}

func (c *Checker) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	params["_meta"] = client.RequestMeta(c.Session, nil)
	return client.CallRaw(ctx, c.Session, method, params)
}

// Run lists the skills and checks them.
func (c *Checker) Run(ctx context.Context) *Report {
	rep := &Report{}
	add := func(name string, status Status, format string, args ...any) {
		rep.Results = append(rep.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
	}

	caps := c.Session.InitializeResult().Capabilities
	settings, declared := caps.Extensions[mcpskills.Extension]
	rep.Declared = declared
	if !declared {
		add("extension declared", Skip, "the server does not declare %s", mcpskills.Extension)
		return rep
	}
	if m, ok := settings.(map[string]any); ok {
		rep.DirectoryRead, _ = m["directoryRead"].(bool)
	}
	if caps.Resources == nil {
		add("resources capability", Fail, "a server declaring %s must declare the resources capability", mcpskills.Extension)
	} else {
		add("resources capability", Pass, "declared")
	}

	entries, ok := c.list(ctx, add)
	if !ok {
		return rep
	}
	for _, e := range entries {
		rep.Skills = append(rep.Skills, c.checkEntry(ctx, e, rep.DirectoryRead, add))
	}

	// Unknown URIs and non-directories must be refused with -32602
	c.expectInvalidParams(ctx, add, "skills/get of an unknown skill", "skills/get", "skill://mcp-tester-no-such-skill/SKILL.md")
	if rep.DirectoryRead {
		c.expectInvalidParams(ctx, add, "directory read of a missing directory", "resources/directory/read", "skill://mcp-tester-no-such-skill")
		if len(entries) > 0 {
			c.expectInvalidParams(ctx, add, "directory read of a file", "resources/directory/read", entries[0].URI)
		}
	}
	return rep
}

// list pages through skills/list.
func (c *Checker) list(ctx context.Context, add func(string, Status, string, ...any)) ([]entry, bool) {
	var entries []entry
	cursor := ""
	for page := 1; ; page++ {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		res, err := c.call(ctx, "skills/list", params)
		if err != nil {
			add("skills/list", Fail, "%v", err)
			return nil, false
		}
		if page == 1 {
			c.checkResultShape(add, "skills/list", res)
		}
		var body struct {
			Skills     []entry `json:"skills"`
			NextCursor string  `json:"nextCursor"`
		}
		if err := remarshal(res, &body); err != nil {
			add("skills/list", Fail, "invalid result: %v", err)
			return nil, false
		}
		entries = append(entries, body.Skills...)
		if body.NextCursor == "" || page >= 100 {
			break
		}
		cursor = body.NextCursor
	}
	add("skills/list", Pass, "%d skill(s)", len(entries))
	return entries, true
}

// checkResultShape checks resultType and the cache hints skills results carry.
func (c *Checker) checkResultShape(add func(string, Status, string, ...any), method string, res map[string]any) {
	if res["resultType"] != "complete" {
		add(method+" resultType", Fail, "%v, must be \"complete\"", res["resultType"])
	}
	_, hasTTL := res["ttlMs"].(float64)
	scope, _ := res["cacheScope"].(string)
	if !hasTTL || (scope != "public" && scope != "private") {
		add(method+" cache hints", Fail, "ttlMs %v, cacheScope %q: both are required", res["ttlMs"], scope)
	}
}

func (c *Checker) checkEntry(ctx context.Context, e entry, directoryRead bool, add func(string, Status, string, ...any)) SkillInfo {
	name, _ := e.Frontmatter["name"].(string)
	info := SkillInfo{URI: e.URI, Name: name}
	info.Description, _ = e.Frontmatter["description"].(string)
	info.License, _ = e.Frontmatter["license"].(string)
	info.AllowedTools, _ = e.Frontmatter["allowed-tools"].(string)
	label := "skill " + e.URI
	var problems []string

	root, isSkillMD := strings.CutSuffix(e.URI, "/SKILL.md")
	if !isSkillMD {
		problems = append(problems, "uri must end with /SKILL.md")
	}
	problems = append(problems, mcpskills.ValidateFrontmatter(e.Frontmatter)...)
	if isSkillMD && path.Base(root) != name {
		problems = append(problems, fmt.Sprintf("last path segment %q must equal the frontmatter name %q", path.Base(root), name))
	}

	// resources: a complete manifest, or "dynamic"
	var manifest []mcpskills.SkillResource
	var dynamic string
	switch {
	case json.Unmarshal(e.Resources, &dynamic) == nil && dynamic == "dynamic":
		info.Dynamic, info.Files = true, -1
		add(label+" integrity", Warn, "resources is \"dynamic\": no digests, the content cannot be verified or bound to an approval")
	case json.Unmarshal(e.Resources, &manifest) == nil && manifest != nil:
		info.Files = len(manifest)
		problems = append(problems, checkManifest(e.URI, root, manifest, &info.Bytes)...)
		if len(manifest) > mcpskills.MaxResources || info.Bytes > mcpskills.MaxTotalSize {
			add(label+" limits", Warn, "%d files / %d bytes exceed %d files / %d bytes; not every host will load it", len(manifest), info.Bytes, mcpskills.MaxResources, mcpskills.MaxTotalSize)
		}
	default:
		problems = append(problems, "resources must be an array or \"dynamic\"; hosts must not load this skill")
	}

	if len(problems) > 0 {
		add(label, Fail, "%s", strings.Join(problems, "; "))
	} else {
		add(label, Pass, "%s: %d file(s), %d bytes", name, info.Files, info.Bytes)
	}

	// skills/get must return the same entry
	if res, err := c.call(ctx, "skills/get", map[string]any{"uri": e.URI}); err != nil {
		add(label+" skills/get", Fail, "%v", err)
	} else {
		c.checkResultShape(add, "skills/get", res)
		var got struct {
			Skill entry `json:"skill"`
		}
		if remarshal(res, &got) != nil || !sameEntry(got.Skill, e) {
			add(label+" skills/get", Fail, "the entry differs from the skills/list entry")
		} else {
			add(label+" skills/get", Pass, "same entry")
		}
	}

	if c.Verify && manifest != nil {
		c.verifyFiles(ctx, label, e, manifest, add)
	}
	if directoryRead && isSkillMD {
		c.checkDirectory(ctx, label, root, manifest, add)
	}
	return info
}

// checkManifest checks completeness and form of a skill's resources array.
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

func (c *Checker) expectInvalidParams(ctx context.Context, add func(string, Status, string, ...any), name, method, uri string) {
	_, err := c.call(ctx, method, map[string]any{"uri": uri})
	var rpcErr *client.RPCError
	switch {
	case err == nil:
		add(name, Fail, "succeeded, must be error -32602")
	case errors.As(err, &rpcErr) && rpcErr.Code == -32602:
		add(name, Pass, "-32602")
	default:
		add(name, Fail, "%v, want error -32602", err)
	}
}

func remarshal(from any, to any) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}

// sameJSON compares two values after a JSON round trip (YAML and JSON
// decoding differ in number types).
func sameJSON(a, b any) bool {
	var x, y any
	if remarshal(a, &x) != nil || remarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func sameEntry(a, b entry) bool {
	var ra, rb any
	_ = json.Unmarshal(a.Resources, &ra)
	_ = json.Unmarshal(b.Resources, &rb)
	return a.URI == b.URI && sameJSON(a.Frontmatter, b.Frontmatter) && reflect.DeepEqual(ra, rb)
}
