package scripting

// assertions.go contains all assertion handlers
import (
	"fmt"
	"strconv"
	"strings"
)

func (r *Runner) handleAssertContains(lineIdx int, line string) error {
	expected := strings.TrimPrefix(line, "assert_contains ")
	expected = strings.Trim(expected, "\"")
	if !strings.Contains(r.lastText, expected) && !strings.Contains(r.lastResponse, expected) {
		return fmt.Errorf("line %d: assertion failed: expected to contain %q", lineIdx+1, expected)
	}
	fmt.Printf("Assertion passed: contains %q\n", expected)
	return nil
}

func (r *Runner) handleAssertEquals(lineIdx int, line string) error {
	expected := strings.TrimPrefix(line, "assert_equals ")
	expected = strings.Trim(expected, "\"")
	if r.lastText != expected && r.lastResponse != expected {
		return fmt.Errorf("line %d: assertion failed: expected exactly %q, but got %q", lineIdx+1, expected, r.lastText)
	}
	fmt.Printf("Assertion passed: equals %q\n", expected)
	return nil
}

func (r *Runner) handleAssertNumber(lineIdx int, line string) error {
	val := strings.TrimSpace(strings.TrimPrefix(line, "assert_number "))
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		return fmt.Errorf("line %d: assertion failed: %q is not a number", lineIdx+1, val)
	}
	fmt.Printf("Assertion passed: %q is a number\n", val)
	return nil
}

func (r *Runner) handleAssertGreaterThan(lineIdx int, line string) error {
	parts := strings.Fields(strings.TrimPrefix(line, "assert_gt "))
	if len(parts) != 2 {
		return fmt.Errorf("line %d: assert_gt expects 2 arguments", lineIdx+1)
	}
	v1, err1 := strconv.ParseFloat(parts[0], 64)
	v2, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("line %d: assert_gt arguments must be numbers", lineIdx+1)
	}
	if v1 <= v2 {
		return fmt.Errorf("line %d: assertion failed: %f is not greater than %f", lineIdx+1, v1, v2)
	}
	fmt.Printf("Assertion passed: %f > %f\n", v1, v2)
	return nil
}
