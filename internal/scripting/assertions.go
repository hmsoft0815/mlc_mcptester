package scripting

// assertions.go contains all assertion handlers
import (
	"fmt"
	"strconv"
	"strings"
)

func (r *Runner) handleAssertContains(lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return err
	}
	return r.handleAssertContainsParts(lineIdx, parts)
}

func (r *Runner) handleAssertContainsParts(lineIdx int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: invalid assert_contains command", lineIdx+1)
	}

	if len(parts) == 2 {
		expected := parts[1]
		if !strings.Contains(r.lastText, expected) && !strings.Contains(r.lastResponse, expected) {
			return fmt.Errorf("line %d: assertion failed: last response does not contain %q", lineIdx+1, expected)
		}
		fmt.Printf("Assertion passed: last response contains %q\n", expected)
	} else if len(parts) >= 3 {
		val1 := parts[1]
		val2 := parts[2]
		if !strings.Contains(val1, val2) {
			return fmt.Errorf("line %d: assertion failed: %q does not contain %q", lineIdx+1, val1, val2)
		}
		fmt.Printf("Assertion passed: %q contains %q\n", val1, val2)
	}
	return nil
}

func (r *Runner) handleAssertEquals(lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return err
	}
	return r.handleAssertEqualsParts(lineIdx, parts)
}

func (r *Runner) handleAssertEqualsParts(lineIdx int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: invalid assert_equals command", lineIdx+1)
	}

	if len(parts) == 2 {
		expected := parts[1]
		if r.lastText != expected && r.lastResponse != expected {
			return fmt.Errorf("line %d: assertion failed: expected exactly %q, but got %q", lineIdx+1, expected, r.lastText)
		}
		fmt.Printf("Assertion passed: last response equals %q\n", expected)
	} else if len(parts) >= 3 {
		val1 := parts[1]
		val2 := parts[2]
		if val1 != val2 {
			return fmt.Errorf("line %d: assertion failed: %q != %q", lineIdx+1, val1, val2)
		}
		fmt.Printf("Assertion passed: %q == %q\n", val1, val2)
	}
	return nil
}

func (r *Runner) handleAssertNumber(lineIdx int, val string) error {
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		return fmt.Errorf("line %d: assertion failed: %q is not a number", lineIdx+1, val)
	}
	fmt.Printf("Assertion passed: %q is a number\n", val)
	return nil
}

func (r *Runner) handleAssertGreaterThan(lineIdx int, s1, s2 string) error {
	v1, err1 := strconv.ParseFloat(s1, 64)
	v2, err2 := strconv.ParseFloat(s2, 64)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("line %d: assert_gt arguments must be numbers", lineIdx+1)
	}
	if v1 <= v2 {
		return fmt.Errorf("line %d: assertion failed: %f is not greater than %f", lineIdx+1, v1, v2)
	}
	fmt.Printf("Assertion passed: %f > %f\n", v1, v2)
	return nil
}

func (r *Runner) handleAssertNumberCommand(lineIdx int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: assert_number expects 1 argument", lineIdx+1)
	}
	return r.handleAssertNumber(lineIdx, parts[1])
}

func (r *Runner) handleAssertGreaterThanCommand(lineIdx int, parts []string) error {
	if len(parts) != 3 {
		return fmt.Errorf("line %d: assert_gt expects 2 arguments", lineIdx+1)
	}
	return r.handleAssertGreaterThan(lineIdx, parts[1], parts[2])
}
