package main

import (
	"fmt"
	"strings"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/httpcheck"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// checkPrompts lists the prompts; a server without prompts costs 20 points.
func (in *inspector) checkPrompts() {
	prompts, err := mcpclient.ListAllPrompts(in.ctx, in.session)
	if err != nil && in.caps.Prompts != nil {
		in.listFailed("prompts/list", err)
	}
	if err != nil {
		return
	}
	in.report.PromptsFound = len(prompts)
	if len(prompts) == 0 {
		in.warn(i18n.T(i18n.MsgNoPrompts))
		in.score -= 20
	} else if format == "text" {
		fmt.Print(i18n.T(i18n.MsgFound, len(prompts), "prompts"))
	}
	for _, p := range prompts {
		in.checkIcons("prompt '"+p.Name+"'", p.Icons)
	}
	in.checkCache("prompts/list")
}

// checkTools lists the tools and rates each of them.
func (in *inspector) checkTools() {
	tools, err := mcpclient.ListAllTools(in.ctx, in.session)
	if err != nil && in.caps.Tools != nil {
		in.listFailed("tools/list", err)
	}
	if err != nil {
		return
	}
	in.report.ToolsFound = len(tools)
	if len(tools) > 0 {
		in.checkToolList(tools)
	}
	in.checkCache("tools/list")
}

func (in *inspector) checkToolList(tools []*mcp.Tool) {
	if format == "text" {
		fmt.Print(i18n.T(i18n.MsgFound, len(tools), "tools"))
	}
	inputSchemaDeduction, safetyBonus := 0, 0
	seen := map[string]bool{}
	for _, t := range tools {
		inputSchemaDeduction += in.checkTool(t, seen)
		// Bonus for safety annotations (readOnlyHint)
		if t.Annotations != nil && t.Annotations.ReadOnlyHint {
			safetyBonus += 2
		}
	}
	in.summarizeOutputSchemas()

	// The spec asks for a deterministic order so clients can cache the list
	if again, err := listToolsAgain(in.ctx, in.session, in.cmdArg, in.urlArg); err == nil && !sameToolOrder(tools, again) {
		in.warn(i18n.T(i18n.MsgToolOrderUnstable))
		in.score -= 5
	}
	in.score += min(safetyBonus, safetyBonusCap) - inputSchemaDeduction
}

// checkTool rates one tool and returns its deduction for a missing input
// schema, which counts outside the categories.
func (in *inspector) checkTool(t *mcp.Tool, seen map[string]bool) (inputSchemaDeduction int) {
	if msg := checkToolName(t.Name); msg != "" {
		in.warn(i18n.T(i18n.MsgInvalidToolName, t.Name, msg))
		in.d.add("toolName", 3)
	}
	if seen[t.Name] {
		in.warn(i18n.T(i18n.MsgDuplicateToolName, t.Name))
		in.d.add("duplicate", 5)
	}
	seen[t.Name] = true
	if t.Title == "" && (t.Annotations == nil || t.Annotations.Title == "") {
		in.warn(i18n.T(i18n.MsgNoTitle, t.Name))
		in.d.add("title", 1)
	}
	if t.Description == "" {
		in.warn(i18n.T(i18n.MsgNoDescription, t.Name))
		in.d.add("description", 5)
	}
	if t.InputSchema == nil {
		in.warn(i18n.T(i18n.MsgNoInputSchema, t.Name))
		inputSchemaDeduction = 10
	} else if !inputSchemaIsObject(t.InputSchema) {
		in.warn(i18n.T(i18n.MsgInputSchemaNotObject, t.Name))
		in.d.add("schemaType", 5)
	}
	in.classifyOutputSchema(t)
	in.checkIcons("tool '"+t.Name+"'", t.Icons)
	if _, problems := httpcheck.XMCPHeaders(t.InputSchema); len(problems) > 0 {
		in.warn(i18n.T(i18n.MsgInvalidXMCPHeader, t.Name, strings.Join(problems, "; ")))
		in.d.add("xMCPHeader", 10)
	}
	return inputSchemaDeduction
}

// classifyOutputSchema counts a declared schema, or notes the tool as exempt
// (--text-only) or as missing one.
func (in *inspector) classifyOutputSchema(t *mcp.Tool) {
	switch {
	case t.OutputSchema != nil:
		in.declaredOutputSchemas++
	case in.textOnly[t.Name]:
		in.exemptTextOnly = append(in.exemptTextOnly, t.Name)
	default:
		in.noOutputSchema = append(in.noOutputSchema, t.Name)
		in.d.add("outputSchema", 1)
	}
}

// summarizeOutputSchemas reports the tools without an output schema, in one
// line or (--hints-per-tool) one per tool, and the exempt ones as INFO.
func (in *inspector) summarizeOutputSchemas() {
	if hintsPerTool {
		for _, name := range in.noOutputSchema {
			in.warn(i18n.T(i18n.MsgNoOutputSchema, name))
		}
	} else if len(in.noOutputSchema) > 0 {
		in.warn(i18n.T(i18n.MsgNoOutputSchemaSummary, len(in.noOutputSchema), strings.Join(in.noOutputSchema, ", ")))
	}
	if len(in.exemptTextOnly) > 0 {
		in.info(i18n.T(i18n.MsgTextOnlyTools, strings.Join(in.exemptTextOnly, ", ")))
		in.report.TextOnlyTools = in.exemptTextOnly
	}
}

// checkResources lists the resources and checks their cache hints.
func (in *inspector) checkResources() {
	resources, err := mcpclient.ListAllResources(in.ctx, in.session)
	if err != nil && in.caps.Resources != nil {
		in.listFailed("resources/list", err)
	}
	if err != nil {
		return
	}
	in.report.ResourcesFound = len(resources)
	if len(resources) > 0 && format == "text" {
		fmt.Print(i18n.T(i18n.MsgFound, len(resources), "resources"))
	}
	for _, r := range resources {
		in.checkIcons("resource '"+r.URI+"'", r.Icons)
	}
	in.checkCache("resources/list")
	in.checkResourceCache(resources)
}

// checkResourceCache: resource contents are typically per user, so "public"
// under credentials can leak them through shared caches.
func (in *inspector) checkResourceCache(resources []*mcp.Resource) {
	switch {
	case !in.modern || len(resources) == 0:
	case readResources:
		uri := resources[0].URI
		if _, scope, ok := in.cacheHints("resources/read", map[string]any{"uri": uri}); !ok {
			in.info(i18n.T(i18n.MsgResourceReadFailed, uri, "no result"))
		} else if scope == "public" && in.authenticated {
			in.warn(i18n.T(i18n.MsgCachePublicResource, uri))
			in.d.add("cacheScope", 10)
		}
	case in.authenticated:
		in.info(i18n.T(i18n.MsgReadResourcesHint))
	}
}

// cacheHints reads a result from the wire and checks its cache hints, part
// of cacheable results since 2026-07-28; the SDK would turn a missing ttlMs
// into 0 and fill cacheScope.
func (in *inspector) cacheHints(method string, params map[string]any) (ttl float64, scope string, ok bool) {
	if params == nil {
		params = map[string]any{}
	}
	params["_meta"] = mcpclient.RequestMeta(in.session, nil)
	result, err := mcpclient.CallRaw(in.ctx, in.session, method, params)
	if err != nil {
		return 0, "", false
	}
	ttl, scope, problems := checkCacheHints(result)
	if len(problems) > 0 {
		in.warn(i18n.T(i18n.MsgCacheHints, method, strings.Join(problems, "; ")))
		in.d.add("cache", 1)
	}
	return ttl, scope, true
}

// checkCache notes a list that is immediately stale or public under credentials.
func (in *inspector) checkCache(method string) {
	if !in.modern {
		return
	}
	ttl, scope, ok := in.cacheHints(method, nil)
	if !ok {
		return
	}
	if ttl == 0 {
		in.staleLists = append(in.staleLists, method)
	}
	if scope == "public" && in.authenticated {
		in.publicLists = append(in.publicLists, method)
	}
}

// reportCache turns the collected cache findings into one INFO line each.
func (in *inspector) reportCache() {
	if len(in.staleLists) > 0 {
		in.info(i18n.T(i18n.MsgCacheStale, strings.Join(in.staleLists, ", ")))
	}
	if len(in.publicLists) > 0 {
		in.info(i18n.T(i18n.MsgCachePublicList, strings.Join(in.publicLists, ", ")))
	}
}
