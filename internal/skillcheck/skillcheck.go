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
	info := infoOf(e)
	label := "skill " + e.URI
	root, isSkillMD := strings.CutSuffix(e.URI, "/SKILL.md")
	problems := identityProblems(e, root, isSkillMD, info.Name)
	manifest, more := checkResources(label, e, root, &info, add)
	problems = append(problems, more...)
	if len(problems) > 0 {
		add(label, Fail, "%s", strings.Join(problems, "; "))
	} else {
		add(label, Pass, "%s: %d file(s), %d bytes", info.Name, info.Files, info.Bytes)
	}

	c.checkGet(ctx, label, e, add)
	if c.Verify && manifest != nil {
		c.verifyFiles(ctx, label, e, manifest, add)
	}
	if directoryRead && isSkillMD {
		c.checkDirectory(ctx, label, root, manifest, add)
	}
	return info
}

// infoOf summarizes an entry from its frontmatter.
func infoOf(e entry) SkillInfo {
	info := SkillInfo{URI: e.URI}
	info.Name, _ = e.Frontmatter["name"].(string)
	info.Description, _ = e.Frontmatter["description"].(string)
	info.License, _ = e.Frontmatter["license"].(string)
	info.AllowedTools, _ = e.Frontmatter["allowed-tools"].(string)
	return info
}

// identityProblems checks the URI, the frontmatter and that the last path
// segment is the skill's name.
func identityProblems(e entry, root string, isSkillMD bool, name string) []string {
	var problems []string
	if !isSkillMD {
		problems = append(problems, "uri must end with /SKILL.md")
	}
	problems = append(problems, mcpskills.ValidateFrontmatter(e.Frontmatter)...)
	if isSkillMD && path.Base(root) != name {
		problems = append(problems, fmt.Sprintf("last path segment %q must equal the frontmatter name %q", path.Base(root), name))
	}
	return problems
}

// checkResources reads resources: a complete manifest, or "dynamic". It
// returns the manifest (nil if there is none) and the problems found.
func checkResources(label string, e entry, root string, info *SkillInfo, add func(string, Status, string, ...any)) ([]mcpskills.SkillResource, []string) {
	var manifest []mcpskills.SkillResource
	var dynamic string
	switch {
	case json.Unmarshal(e.Resources, &dynamic) == nil && dynamic == "dynamic":
		info.Dynamic, info.Files = true, -1
		add(label+" integrity", Warn, "resources is \"dynamic\": no digests, the content cannot be verified or bound to an approval")
		return nil, nil
	case json.Unmarshal(e.Resources, &manifest) == nil && manifest != nil:
		info.Files = len(manifest)
		problems := checkManifest(e.URI, root, manifest, &info.Bytes)
		if len(manifest) > mcpskills.MaxResources || info.Bytes > mcpskills.MaxTotalSize {
			add(label+" limits", Warn, "%d files / %d bytes exceed %d files / %d bytes; not every host will load it", len(manifest), info.Bytes, mcpskills.MaxResources, mcpskills.MaxTotalSize)
		}
		return manifest, problems
	default:
		return nil, []string{"resources must be an array or \"dynamic\"; hosts must not load this skill"}
	}
}

// checkGet: skills/get must return the same entry as skills/list.
func (c *Checker) checkGet(ctx context.Context, label string, e entry, add func(string, Status, string, ...any)) {
	res, err := c.call(ctx, "skills/get", map[string]any{"uri": e.URI})
	if err != nil {
		add(label+" skills/get", Fail, "%v", err)
		return
	}
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
