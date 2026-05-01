package claude

import (
	"ds2api/internal/toolcall"
	"fmt"
	"time"

	"ds2api/internal/util"
)

func BuildMessageResponse(messageID, model string, normalizedMessages []any, finalThinking, finalText string, toolNames []string, refFileIDs []string) map[string]any {
	detected := toolcall.ParseToolCalls(finalText, toolNames)
	if len(detected) == 0 && finalText == "" && finalThinking != "" {
		detected = toolcall.ParseToolCalls(finalThinking, toolNames)
	}
	content := make([]map[string]any, 0, 4)
	if finalThinking != "" {
		content = append(content, map[string]any{"type": "thinking", "thinking": finalThinking})
	}
	stopReason := "end_turn"
	if len(detected) > 0 {
		stopReason = "tool_use"
		for i, tc := range detected {
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    fmt.Sprintf("toolu_%d_%d", time.Now().Unix(), i),
				"name":  tc.Name,
				"input": tc.Input,
			})
		}
	} else {
		if finalText == "" {
			finalText = "抱歉，没有生成有效的响应内容。"
		}
		content = append(content, map[string]any{"type": "text", "text": finalText})
	}
	return map[string]any{
		"id":            messageID,
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  util.EstimateInputTokens(renderMessages(normalizedMessages), refFileIDs),
			"output_tokens": util.EstimateTokens(finalThinking) + util.EstimateTokens(finalText),
		},
	}
}

// renderMessages converts []any to []map[string]any and renders the prompt
// text for token counting. This avoids the inaccurate fmt.Sprintf("%v") approach.
func renderMessages(messages []any) string {
	msgMaps := make([]map[string]any, 0, len(messages))
	for _, item := range messages {
		if m, ok := item.(map[string]any); ok {
			msgMaps = append(msgMaps, m)
		}
	}
	return util.MessagesPrepare(msgMaps)
}
