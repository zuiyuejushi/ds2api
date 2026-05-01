package toolcall

import (
	"testing"

	"ds2api/internal/httpapi/openai/shared"
)

func TestCleanVisibleOutputWithDSML(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	result := shared.CleanVisibleOutput(input, false)

	t.Logf("Original: %s", input)
	t.Logf("Cleaned:  %s", result)

	// 检查清理后的内容是否还能被解析
	parsed := ParseStandaloneToolCalls(result, nil)
	if len(parsed) == 0 {
		t.Errorf("Cleaned content could not be parsed as tool calls")
	} else {
		t.Logf("Parsed %d tool calls", len(parsed))
	}
}
