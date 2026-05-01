package toolcall

import (
	"encoding/json"
	"testing"
)

func TestParseStandaloneToolCallsWithDSML(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	result := ParseStandaloneToolCalls(input, nil)

	if len(result) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(result))
	}

	if result[0].Name != "read" {
		t.Errorf("Expected name 'read', got '%s'", result[0].Name)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Parsed result:\n%s", string(data))
}

func TestParseStandaloneToolCallsDetailedWithDSML(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	result := ParseStandaloneToolCallsDetailed(input, nil)

	if !result.SawToolCallSyntax {
		t.Errorf("Expected SawToolCallSyntax to be true")
	}

	if len(result.Calls) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if len(result.Calls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(result.Calls))
	}

	if result.Calls[0].Name != "read" {
		t.Errorf("Expected name 'read', got '%s'", result.Calls[0].Name)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Parsed result:\n%s", string(data))
}
