package toolcall

import (
	"encoding/json"
	"testing"
)

func TestDSMLFullWidthParsing(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls> <｜｜DSML｜｜invoke name="read"> <｜｜DSML｜｜parameter name="filePath" string="true">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend/src/services/battleService.ts</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name="limit" string="false">100</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> <｜｜DSML｜｜invoke name="read"> <｜｜DSML｜｜parameter name="filePath" string="true">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend/prisma/seed.ts</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name="limit" string="false">80</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> <｜｜DSML｜｜invoke name="read"> <｜｜DSML｜｜parameter name="filePath" string="true">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend/src/routes/rune.ts</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name="limit" string="false">60</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> </｜｜DSML｜｜tool_calls>`

	result := ParseToolCalls(input, nil)

	if len(result) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tool calls, got %d", len(result))
	}

	for i, tc := range result {
		if tc.Name != "read" {
			t.Errorf("Call %d: expected name 'read', got '%s'", i, tc.Name)
		}
		if tc.Input == nil {
			t.Errorf("Call %d: expected non-nil input", i)
			continue
		}
		if _, ok := tc.Input["filePath"]; !ok {
			t.Errorf("Call %d: missing filePath parameter", i)
		}
		// Read tool parameters are simplified - only filePath is kept, limit is removed
		if len(tc.Input) != 1 {
			t.Errorf("Call %d: expected 1 parameter (filePath), got %d", i, len(tc.Input))
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Parsed result:\n%s", string(data))
}

func TestDSMLHalfWidthParsing(t *testing.T) {
	input := `||DSML||<tool_calls>||DSML||<invoke name="read"><parameter name="filePath">/test/file.ts</parameter></invoke></tool_calls>`

	result := ParseToolCalls(input, nil)

	if len(result) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if result[0].Name != "read" {
		t.Errorf("Expected name 'read', got '%s'", result[0].Name)
	}
}

func TestDSMLWithWhitespace(t *testing.T) {
	input := `  ｜｜DSML｜｜  <tool_calls>  ｜｜DSML｜｜  <invoke name="test"><parameter name="key">value</parameter></invoke>  </tool_calls>`

	result := ParseToolCalls(input, nil)

	if len(result) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if result[0].Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", result[0].Name)
	}
}
