package scripting

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/hmsoft0815/mlc_mcptester/internal/skillcheck"
)

// handleListSkillsCommand runs "list_skills": the skills are stored as
// {skills: [...]}, so set_var can address skills.0.name; the text lists the names.
func (r *Runner) handleListSkillsCommand(ctx context.Context, i int) error {
	rep := (&skillcheck.Checker{Session: r.session}).Run(ctx)
	if !rep.Declared {
		return fmt.Errorf("line %d: the server does not declare the skills extension", i+1)
	}
	skills := make([]any, len(rep.Skills))
	names := make([]string, len(rep.Skills))
	for j, s := range rep.Skills {
		skills[j] = map[string]any{"uri": s.URI, "name": s.Name, "description": s.Description, "files": s.Files, "dynamic": s.Dynamic}
		names[j] = s.Name
	}
	r.updateState(map[string]any{"skills": skills, "count": len(skills)}, strings.Join(names, "\n"))
	fmt.Fprintf(r.w(), "Skills: %s\n", strings.Join(names, ", "))
	return nil
}

// handleVerifySkillsCommand runs "verify_skills": all skill checks including
// reading every file; any FAIL fails the command.
func (r *Runner) handleVerifySkillsCommand(ctx context.Context, i int) error {
	rep := (&skillcheck.Checker{Session: r.session, Verify: true}).Run(ctx)
	var fails []string
	for _, res := range rep.Results {
		if res.Status == skillcheck.Fail {
			fails = append(fails, res.Name+": "+res.Detail)
		}
	}
	if len(fails) > 0 {
		return fmt.Errorf("line %d: skills verification failed: %s", i+1, strings.Join(fails, "; "))
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, fmt.Sprintf("%d skill(s) verified", len(rep.Skills))))
	return nil
}
