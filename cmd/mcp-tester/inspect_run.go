package main

import (
	"cmp"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/hmsoft0815/mlc_mcptester/internal/skillcheck"
	"github.com/hmsoft0815/mlc_mcptester/internal/taskcheck"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// inspector carries one inspect run: the connection, what it found, the
// deductions and the lines it reports.
type inspector struct {
	ctx            context.Context
	session        *mcp.ClientSession
	cmdArg, urlArg string          // what was connected, for the title and a second connection
	textOnly       map[string]bool // tools exempt from the output schema check

	init          *mcp.InitializeResult
	caps          *mcp.ServerCapabilities
	modern        bool // protocol 2026-07-28 or later
	authenticated bool // connected with credentials

	report InspectionReport
	score  int // 100, minus deductions without a category, plus the safety bonus
	d      *deductions

	recommendations, protocolErrors, infos []string
	// inspect never calls a tool — it cannot know which ones are free of
	// side effects — so a declared schema is all it can see, not whether the
	// results honour it.
	declaredOutputSchemas int
	// Missing output schemas are collected so they cost one line, not one per tool
	noOutputSchema, exemptTextOnly []string
	// Collected per finding, so the INFO lines name all lists at once
	staleLists, publicLists []string
}

func newInspector(ctx context.Context, session *mcp.ClientSession, cmdArg, urlArg string, textOnly map[string]bool) *inspector {
	return &inspector{
		ctx: ctx, session: session, cmdArg: cmdArg, urlArg: urlArg, textOnly: textOnly,
		authenticated: urlArg != "" && (bearerToken != "" || len(headerFlags) > 0 ||
			oauthEnabled || oauthClientCredentials || oauthEnterprise),
		report:          InspectionReport{LatestProtocolVersion: latestProtocolRevision},
		score:           100,
		d:               newDeductions(inspectDeductionCaps()),
		recommendations: []string{},
		protocolErrors:  []string{},
		infos:           []string{},
	}
}

func (in *inspector) warn(msg string) { in.recommendations = append(in.recommendations, msg) }
func (in *inspector) info(msg string) { in.infos = append(in.infos, msg) }

func (in *inspector) listFailed(method string, err error) {
	in.protocolErrors = append(in.protocolErrors, i18n.T(i18n.MsgListFailed, method, err))
	in.d.add("listFailure", listFailurePenalty)
}

// run performs all checks and settles the score.
func (in *inspector) run() {
	in.readServerInfo()
	if format == "text" {
		in.printHeader()
	}
	in.checkIcons("server "+in.report.ServerName, in.init.ServerInfo.Icons)
	in.checkPrompts()
	in.checkTools()
	in.checkResources()
	in.reportCache()
	in.checkExtensions()
	in.checkDeprecated()

	in.report.Score = finalScore(in.score, in.d)
	in.report.Recommendations = in.recommendations
	in.report.Errors = in.protocolErrors
	in.report.Infos = in.infos
}

// readServerInfo takes name, version, protocol and extensions from the
// handshake and rates the protocol revision.
func (in *inspector) readServerInfo() {
	in.init = in.session.InitializeResult()
	r := &in.report
	r.ServerName = in.init.ServerInfo.Name
	r.ServerVersion = in.init.ServerInfo.Version
	r.ProtocolVersion = in.init.ProtocolVersion
	r.HasInstructions = in.init.Instructions != ""
	in.modern = r.ProtocolVersion >= latestProtocolRevision

	switch behind := revisionsBehind(r.ProtocolVersion); {
	case behind < 0:
		in.warn(i18n.T(i18n.MsgUnknownProtocol, r.ProtocolVersion))
		in.d.add("protocol", 10)
	case behind > 0:
		in.warn(i18n.T(i18n.MsgOutdatedProtocol, r.ProtocolVersion, behind, latestProtocolRevision))
		in.d.add("protocol", 10*behind)
	}

	in.caps = in.init.Capabilities
	for name := range in.caps.Extensions {
		r.Extensions = append(r.Extensions, name)
	}
	sort.Strings(r.Extensions)
}

// printHeader prints the title, the server and its capabilities (text format).
func (in *inspector) printHeader() {
	// The profile name, or what was connected without one
	target := profile
	if target == "" {
		target = cmp.Or(in.urlArg, in.cmdArg)
	}
	fmt.Print(i18n.T(i18n.MsgInspectionTitle, target))
	fmt.Print(i18n.T(i18n.MsgServerInfo, in.report.ServerName, in.report.ServerVersion))
	fmt.Print(i18n.T(i18n.MsgProtocolVersion, in.report.ProtocolVersion))
	fmt.Println(i18n.T(i18n.MsgCapabilities))
	in.printCapabilities()
}

func (in *inspector) printCapabilities() {
	caps := in.caps
	fmt.Print(i18n.T(i18n.MsgTools, caps.Tools != nil))
	if caps.Tools != nil && caps.Tools.ListChanged {
		fmt.Print(i18n.T(i18n.MsgSubscription))
	}
	fmt.Println()

	fmt.Print(i18n.T(i18n.MsgPrompts, caps.Prompts != nil))
	if caps.Prompts != nil && caps.Prompts.ListChanged {
		fmt.Print(i18n.T(i18n.MsgSubscription))
	}
	fmt.Println()

	fmt.Print(i18n.T(i18n.MsgResources, caps.Resources != nil))
	fmt.Println()

	fmt.Print(i18n.T(i18n.MsgCompletions, caps.Completions != nil))
	//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577) but still used by servers and regular before; remove with T-20260927-05
	fmt.Print(i18n.T(i18n.MsgLogging, caps.Logging != nil))
	fmt.Print(i18n.T(i18n.MsgProgress))
	fmt.Print(i18n.T(i18n.MsgCancel))
	if in.report.HasInstructions {
		fmt.Print(i18n.T(i18n.MsgInstructions, len(in.init.Instructions)))
	} else {
		fmt.Print(i18n.T(i18n.MsgNoInstructions))
	}
	if len(in.report.Extensions) > 0 {
		fmt.Print(i18n.T(i18n.MsgExtensions, strings.Join(in.report.Extensions, ", ")))
	}
}

func (in *inspector) checkIcons(owner string, icons []mcp.Icon) {
	for _, icon := range icons {
		if msg := checkIconSource(icon.Source); msg != "" {
			in.warn(i18n.T(i18n.MsgInvalidIcon, owner, msg, icon.Source))
			in.d.add("icon", 5)
		}
	}
}

// checkExtensions rates the extensions inspect can check without a tool call.
func (in *inspector) checkExtensions() {
	// Skills are instructions that reach the model: count them, details via 'skills'
	if _, ok := in.caps.Extensions[mcpskills.Extension]; ok {
		skillsReport := (&skillcheck.Checker{Session: in.session}).Run(in.ctx)
		if format == "text" {
			fmt.Print(i18n.T(i18n.MsgFound, len(skillsReport.Skills), "skills"))
		}
		if skillsReport.Failed() {
			in.warn(i18n.T(i18n.MsgSkillsInvalid))
			in.d.add("skills", 10)
		}
	}

	// Tasks: only the tasks/* error codes, no tool runs; the life of a
	// task needs a tool chosen by the user ('tasks --tool')
	if _, ok := in.caps.Extensions[mcptasks.Extension]; ok {
		if failed := (&taskcheck.Checker{Session: in.session}).Run(in.ctx).FailedNames(); len(failed) > 0 {
			in.warn(i18n.T(i18n.MsgTasksInvalid, strings.Join(failed, ", ")))
			in.d.add("tasks", 10)
		}
	}
}

// checkDeprecated: logging is deprecated as of 2026-07-28 (SEP-2577). No
// deduction either way, but a server that still declares it should know that
// support is running out. Older revisions keep it as a regular feature.
func (in *inspector) checkDeprecated() {
	//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577) but still used by servers and regular before; remove with T-20260927-05
	if in.modern && in.caps.Logging != nil {
		in.info(i18n.T(i18n.MsgDeprecatedLogging))
	}
}
