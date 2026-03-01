package scripting

import (
	"testing"
)

func TestAssertionHandlers(t *testing.T) {
	r := &Runner{}

	t.Run("handleAssertContains", func(t *testing.T) {
		r.lastResponse = "Hello World"
		err := r.handleAssertContains(0, "assert_contains \"Hello\"")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = r.handleAssertContains(0, "assert_contains \"Universe\"")
		if err == nil {
			t.Errorf("expected error for missing content, got nil")
		}
	})

	t.Run("handleAssertEquals", func(t *testing.T) {
		r.lastResponse = "Exact match"
		err := r.handleAssertEquals(0, "assert_equals \"Exact match\"")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = r.handleAssertEquals(0, "assert_equals \"Wrong\"")
		if err == nil {
			t.Errorf("expected error for mismatch, got nil")
		}
	})

	t.Run("handleAssertNumber", func(t *testing.T) {
		err := r.handleAssertNumber(0, "123.45")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = r.handleAssertNumber(0, "not-a-number")
		if err == nil {
			t.Errorf("expected error for non-number, got nil")
		}
	})

	t.Run("handleAssertGreaterThan", func(t *testing.T) {
		err := r.handleAssertGreaterThan(0, "10", "5")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = r.handleAssertGreaterThan(0, "5", "10")
		if err == nil {
			t.Errorf("expected error for 5 > 10, got nil")
		}

		err = r.handleAssertGreaterThan(0, "10", "10")
		if err == nil {
			t.Errorf("expected error for 10 > 10, got nil")
		}

		err = r.handleAssertGreaterThan(0, "inv", "10")
		if err == nil {
			t.Errorf("expected error for invalid input, got nil")
		}
	})

	t.Run("handleSetVar", func(t *testing.T) {
		r.variables = make(map[string]string)
		r.lastResponse = `{"data": {"value": 42}, "status": "ok"}`
		r.lastRawMap = map[string]any{"dummy": "data"} // Ensure not nil

		// Test direct raw response
		err := r.handleSetVar(0, "set_var my_val rawResponse")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if r.variables["my_val"] != r.lastResponse {
			t.Errorf("expected %s, got %s", r.lastResponse, r.variables["my_val"])
		}

		// Test structured content (simulated)
		r.lastRawMap = map[string]any{
			"data": map[string]any{
				"value": 42.0,
			},
		}
		err = r.handleSetVar(0, "set_var deep_val structuredContent.data.value")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if r.variables["deep_val"] != "42" {
			t.Errorf("expected 42, got %s", r.variables["deep_val"])
		}

		// Test invalid path
		err = r.handleSetVar(0, "set_var err_val structuredContent.nonexistent")
		if err == nil {
			t.Errorf("expected error for invalid path, got nil")
		}

		err = r.handleSetVar(0, "set_var missing_args")
		if err == nil {
			t.Errorf("expected error for missing arguments, got nil")
		}
	})
	t.Run("handleAssertStringLength", func(t *testing.T) {
		err := r.handleAssertStringLength(0, "hello", 3, 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = r.handleAssertStringLength(0, "h", 3, 10)
		if err == nil {
			t.Errorf("expected error for too short string, got nil")
		}

		err = r.handleAssertStringLength(0, "this is a very long string", 3, 10)
		if err == nil {
			t.Errorf("expected error for too long string, got nil")
		}
	})
}
