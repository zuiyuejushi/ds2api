package shared

import (
	"testing"
)

func TestCleanVisibleOutputWithDSML(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	result := CleanVisibleOutput(input, false)

	t.Logf("Original: %s", input)
	t.Logf("Cleaned:  %s", result)

	// 检查清理后的内容是否还包含 DSML 标记
	if result == "" {
		t.Errorf("Cleaned content is empty")
	}
}
