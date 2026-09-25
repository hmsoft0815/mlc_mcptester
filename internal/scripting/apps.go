package scripting

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/appcheck"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
)

// handleVerifyAppsCommand runs "verify_apps": the MCP Apps checks; any FAIL
// fails the command. The app list is stored as {apps: [...], count}.
func (r *Runner) handleVerifyAppsCommand(ctx context.Context, i int) error {
	rep := appcheck.Run(ctx, r.session)
	apps := make([]any, len(rep.Apps))
	uris := make([]string, len(rep.Apps))
	for j, a := range rep.Apps {
		apps[j] = map[string]any{"uri": a.URI, "tools": a.Tools, "permissions": a.Permissions}
		uris[j] = a.URI
	}
	r.updateState(map[string]any{"apps": apps, "count": len(apps)}, strings.Join(uris, "\n"))
	var fails []string
	for _, res := range rep.Results {
		if res.Status == appcheck.Fail {
			fails = append(fails, res.Name+": "+res.Detail)
		}
	}
	if len(fails) > 0 {
		return fmt.Errorf("line %d: MCP Apps verification failed: %s", i+1, strings.Join(fails, "; "))
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, fmt.Sprintf("%d app(s) verified", len(rep.Apps))))
	return nil
}
