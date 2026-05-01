package toolcall

import (
	"encoding/json"
	"testing"
)

func TestDSMLNoSpace(t *testing.T) {
	// 测试 DSML 标记和 tool_calls 之间没有空格的情况
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	result := ParseToolCalls(input, nil)

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

func TestDSMLWithSpace(t *testing.T) {
	// 测试 DSML 标记和 tool_calls 之间有空格的情况
	input := `<｜｜DSML｜｜ tool_calls><｜｜DSML｜｜ invoke name="read"><｜｜DSML｜｜ parameter name="filePath">test.txt</｜｜DSML｜｜ parameter></｜｜DSML｜｜ invoke></｜｜DSML｜｜ tool_calls>`

	result := ParseToolCalls(input, nil)

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
