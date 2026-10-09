package taskcheck

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

var validStatus = map[string]bool{"working": true, "input_required": true, "completed": true, "failed": true, "cancelled": true}

var terminal = map[string]bool{"completed": true, "failed": true, "cancelled": true}

// checkTask checks a Task as it arrives in a CreateTaskResult (resultType
// "task"), a tasks/get result (resultType "complete") or a notifications/tasks
// notification (resultType "", carrying the details like tasks/get). It
// returns the violated MUSTs and SHOULDs.
func checkTask(t map[string]any, resultType string) (must, should []string) {
	if resultType != "" && t["resultType"] != resultType {
		must = append(must, fmt.Sprintf("resultType %v, must be %q", t["resultType"], resultType))
	}
	if id, _ := t["taskId"].(string); id == "" {
		must = append(must, "taskId missing or empty")
	}
	status, _ := t["status"].(string)
	if !validStatus[status] {
		must = append(must, fmt.Sprintf("status %q is not one of working, input_required, completed, failed, cancelled", status))
	}
	for _, key := range []string{"createdAt", "lastUpdatedAt"} {
		if p := checkTimestamp(t, key); p != "" {
			must = append(must, p)
		}
	}
	if ttl, ok := t["ttlMs"]; !ok {
		must = append(must, "ttlMs missing (an integer, or null for unlimited)")
	} else if ttl != nil && !isNonNegativeInt(ttl) {
		must = append(must, fmt.Sprintf("ttlMs %v is not a non-negative integer or null", ttl))
	}
	if poll, ok := t["pollIntervalMs"]; ok && !isNonNegativeInt(poll) {
		must = append(must, fmt.Sprintf("pollIntervalMs %v is not a non-negative integer", poll))
	}
	if resultType != "task" {
		m, s := checkDetail(t, status)
		must, should = append(must, m...), append(should, s...)
	}
	return must, should
}

// checkDetail checks the status-specific fields of a tasks/get result.
func checkDetail(t map[string]any, status string) (must, should []string) {
	switch status {
	case "input_required":
		if reqs, _ := t["inputRequests"].(map[string]any); len(reqs) == 0 {
			must = append(must, "input_required without inputRequests")
		}
	case "completed":
		if _, ok := t["result"].(map[string]any); !ok {
			must = append(must, "completed without a result object")
		}
	case "failed":
		e, _ := t["error"].(map[string]any)
		if _, ok := e["code"].(float64); !ok {
			must = append(must, "failed without a JSON-RPC error (error.code)")
		}
		if msg, _ := t["statusMessage"].(string); msg == "" {
			should = append(should, "failed without statusMessage")
		}
	}
	return must, should
}

func checkTimestamp(t map[string]any, key string) string {
	s, ok := t[key].(string)
	if !ok || s == "" {
		return key + " missing"
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
		return fmt.Sprintf("%s %q is not an ISO 8601 timestamp", key, s)
	}
	return ""
}

func isNonNegativeInt(v any) bool {
	f, ok := v.(float64)
	return ok && f >= 0 && f == math.Trunc(f)
}

var simpleID = regexp.MustCompile(`^[0-9]+$`)

// guessableID reports a task id that is likely enumerable: task ids may act
// as bearer tokens and MUST carry enough entropy.
func guessableID(id string) string {
	switch {
	case simpleID.MatchString(id):
		return "consists of digits only (a counter?)"
	case len(id) < 16:
		return fmt.Sprintf("has only %d characters", len(id))
	}
	return ""
}
