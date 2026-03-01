package scripting

// assertions.go contains all assertion handlers
// can be called with assert_contains, assert_equals, assert_gt, assert_number
// maybe we enhance this to support more assertions, but for now this is enough
import (
	"fmt"
	"strconv"
	"strings"
)

// assert_contains <value>
func (r *Runner) handleAssertContains(lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return err
	}
	return r.handleAssertContainsParts(lineIdx, parts)
}

// assert_contains <value>
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

// assert_equals <value1> <value2>
func (r *Runner) handleAssertEquals(lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return err
	}
	return r.handleAssertEqualsParts(lineIdx, parts)
}

// assert_equals <value1> <value2>
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

// assert_number <value>
func (r *Runner) handleAssertNumber(lineIdx int, val string) error {
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		return fmt.Errorf("line %d: assertion failed: %q is not a number", lineIdx+1, val)
	}
	fmt.Printf("Assertion passed: %q is a number\n", val)
	return nil
}

// assert_gt <value1> <value2>
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

// assert_number <value>
func (r *Runner) handleAssertNumberCommand(lineIdx int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: assert_number expects 1 argument", lineIdx+1)
	}
	return r.handleAssertNumber(lineIdx, parts[1])
}

// assert_gt <value1> <value2>
func (r *Runner) handleAssertGreaterThanCommand(lineIdx int, parts []string) error {
	if len(parts) != 3 {
		return fmt.Errorf("line %d: assert_gt expects 2 arguments", lineIdx+1)
	}
	return r.handleAssertGreaterThan(lineIdx, parts[1], parts[2])
}

// assert_string_length <value> <min> <max>
func (r *Runner) handleAssertStringLengthCommand(lineIdx int, parts []string) error {
	if len(parts) < 4 {
		return fmt.Errorf("line %d: assert_string_length expects at least 3 arguments (value, min, max), got %d", lineIdx+1, len(parts)-1)
	}
	// The value might have been split if it contained spaces and wasn't quoted
	// or if it was interpolated. Everything except the last two parts is the value.
	maxIdx := len(parts) - 1
	minIdx := len(parts) - 2
	valParts := parts[1:minIdx]
	val := strings.Join(valParts, " ")

	min, err1 := strconv.Atoi(parts[minIdx])
	max, err2 := strconv.Atoi(parts[maxIdx])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("line %d: assert_string_length min and max must be integers", lineIdx+1)
	}
	return r.handleAssertStringLength(lineIdx, val, min, max)
}

// string length min,max ?
func (r *Runner) handleAssertStringLength(lineIdx int, s string, min, max int) error {
	if len(s) < min || len(s) > max {
		return fmt.Errorf("line %d: assertion failed: string length %d is not between %d and %d", lineIdx+1, len(s), min, max)
	}
	fmt.Printf("Assertion passed: string length %d is between %d and %d\n", len(s), min, max)
	return nil
}
