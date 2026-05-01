package toolstream

import (
	"testing"
)

func TestProcessChunkWithDSML(t *testing.T) {
	input := `<｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	var state State
	events := ProcessChunk(&state, input, nil)
	events = append(events, Flush(&state, nil)...)

	// 检查是否有 ToolCalls 事件
	hasToolCalls := false
	for _, evt := range events {
		if len(evt.ToolCalls) > 0 {
			hasToolCalls = true
			t.Logf("Found ToolCalls: %+v", evt.ToolCalls)
		}
	}

	if !hasToolCalls {
		t.Errorf("Expected ToolCalls event, got none. Events: %+v", events)
	}
}

func TestProcessChunkWithDSMLAndPrefix(t *testing.T) {
	// 测试带前缀文本的情况
	input := `Some prefix text <｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name="read"><｜｜DSML｜｜parameter name="filePath">test.txt</｜｜DSML｜｜parameter></｜｜DSML｜｜invoke></｜｜DSML｜｜tool_calls>`

	var state State
	events := ProcessChunk(&state, input, nil)
	events = append(events, Flush(&state, nil)...)

	// 检查是否有 ToolCalls 事件
	hasToolCalls := false
	hasPrefix := false
	for _, evt := range events {
		if len(evt.ToolCalls) > 0 {
			hasToolCalls = true
			t.Logf("Found ToolCalls: %+v", evt.ToolCalls)
		}
		if evt.Content != "" {
			hasPrefix = true
			t.Logf("Found Content: %s", evt.Content)
		}
	}

	if !hasToolCalls {
		t.Errorf("Expected ToolCalls event, got none. Events: %+v", events)
	}
	if !hasPrefix {
		t.Errorf("Expected prefix content, got none")
	}
}
