package history

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"ds2api/internal/auth"
	dsclient "ds2api/internal/deepseek/client"
	"ds2api/internal/httpapi/openai/shared"
	"ds2api/internal/promptcompat"
)

const (
	currentInputFilename    = "INPUT.txt"
	currentInputContentType = "text/plain; charset=utf-8"
	currentInputPurpose     = "assistants"
)

// Internal system prompt content that should be excluded from INPUT.txt
// This is the prompt added by buildConversationContinuityInstructions
var internalSystemPromptPatterns = []string{
	"Continue the conversation from the full prior context and the latest tool results.",
	"Treat earlier messages as binding context; answer the user's current request as a continuation, not a restart.",
	"Keep reasoning internal. Do not leave the final user-facing answer only in reasoning; always provide the answer in visible assistant content.",
}

// Internal prompt markers that should be stripped from the uploaded content
var internalPromptMarkers = []*regexp.Regexp{
	regexp.MustCompile(`<｜begin▁of▁sentence｜>`),
	regexp.MustCompile(`<｜System｜>`),
	regexp.MustCompile(`<｜User｜>`),
	regexp.MustCompile(`<｜Assistant｜>`),
	regexp.MustCompile(`<｜Tool｜>`),
	regexp.MustCompile(`<｜end▁of▁sentence｜>`),
	regexp.MustCompile(`<｜end▁of▁toolresults｜>`),
	regexp.MustCompile(`<｜end▁of▁instructions｜>`),
	regexp.MustCompile(`\d{13}`), // Millisecond timestamps
}

// CurrentInputSplitService handles splitting the current user input into a file
// when it exceeds the model's context limit.
type CurrentInputSplitService struct {
	Store shared.ConfigReader
	DS    shared.DeepSeekCaller
}

// Apply uploads the Coding Agent prompt + user input as a file unconditionally.
// This excludes our internal system prompts (conversation continuity instructions).
// This should be called after history split to ensure we're only processing the current turn.
func (s CurrentInputSplitService) Apply(ctx context.Context, a *auth.RequestAuth, stdReq promptcompat.StandardRequest) (promptcompat.StandardRequest, error) {
	if s.DS == nil || s.Store == nil || a == nil {
		return stdReq, nil
	}

	// Build content from all messages (excluding history which was already split)
	// This includes Coding Agent's system prompt and user input
	content := buildCurrentTurnContent(stdReq.Messages)
	if strings.TrimSpace(content) == "" {
		return stdReq, nil
	}

	// Strip internal system prompts (our conversation continuity instructions)
	content = stripInternalSystemPrompts(content)

	// Strip internal markers
	content = stripInternalMarkers(content)

	// Build the transcript
	inputText := buildCurrentInputTranscript(content)
	if strings.TrimSpace(inputText) == "" {
		return stdReq, errors.New("current input split produced empty transcript")
	}

	// Upload the current input as a file
	result, err := s.DS.UploadFile(ctx, a, dsclient.UploadFileRequest{
		Filename:    currentInputFilename,
		ContentType: currentInputContentType,
		Purpose:     currentInputPurpose,
		Data:        []byte(inputText),
	}, 3)
	if err != nil {
		return stdReq, fmt.Errorf("upload current input file: %w", err)
	}

	fileID := strings.TrimSpace(result.ID)
	if fileID == "" {
		return stdReq, errors.New("upload current input file returned empty file id")
	}

	// Find the last user message index to replace
	_, lastUserIndex := extractLastUserMessage(stdReq.Messages)
	if lastUserIndex < 0 {
		return stdReq, nil
	}

	// Build replacement message with clear priority indication
	// If there's history, we need to guide the model to prioritize current input
	replacementContent := buildCurrentInputPrompt(stdReq.RefFileIDs, fileID)

	replacementMsg := map[string]any{
		"role":    "user",
		"content": replacementContent,
	}

	// Create new messages slice with the replacement
	newMessages := make([]any, len(stdReq.Messages))
	copy(newMessages, stdReq.Messages)
	newMessages[lastUserIndex] = replacementMsg

	// Update the request
	stdReq.Messages = newMessages
	stdReq.RefFileIDs = prependUniqueRefFileID(stdReq.RefFileIDs, fileID)
	stdReq.FinalPrompt, stdReq.ToolNames = promptcompat.BuildOpenAIPrompt(newMessages, stdReq.ToolsRaw, "", stdReq.ToolChoice, stdReq.Thinking)

	return stdReq, nil
}

// buildCurrentTurnContent builds content from all current turn messages
// (excluding history messages that were already split to HISTORY.txt)
func buildCurrentTurnContent(messages []any) string {
	var parts []string
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content := extractMessageContent(m)
		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// stripInternalSystemPrompts removes our internal conversation continuity instructions
func stripInternalSystemPrompts(content string) string {
	for _, pattern := range internalSystemPromptPatterns {
		content = strings.ReplaceAll(content, pattern, "")
	}
	return strings.TrimSpace(content)
}

// stripInternalMarkers removes internal prompt markers
func stripInternalMarkers(content string) string {
	for _, re := range internalPromptMarkers {
		content = re.ReplaceAllString(content, "")
	}
	return strings.TrimSpace(content)
}

// extractLastUserMessage finds the last user message in the messages slice
func extractLastUserMessage(messages []any) (map[string]any, int) {
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(shared.AsString(msg["role"])))
		if role == "user" {
			return msg, i
		}
	}
	return nil, -1
}

// extractMessageContent extracts the text content from a message
func extractMessageContent(msg map[string]any) string {
	content := msg["content"]
	if content == nil {
		return ""
	}

	switch v := content.(type) {
	case string:
		return v
	case []any:
		// Handle array content (e.g., multimodal messages)
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				typeStr := strings.ToLower(strings.TrimSpace(shared.AsString(m["type"])))
				if typeStr == "text" {
					text := shared.AsString(m["text"])
					if text != "" {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return shared.AsString(content)
	}
}

// buildCurrentInputTranscript builds the transcript content for the current input file
func buildCurrentInputTranscript(content string) string {
	var sb strings.Builder
	sb.WriteString("[当前用户输入 - Current User Input]\n")
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\n" + strings.Repeat("=", 51) + "\n")
	sb.WriteString("[输入结束 - End of Input]")
	return sb.String()
}

// buildCurrentInputPrompt builds the prompt that references the current input file
// It ensures the model prioritizes the current input over history
func buildCurrentInputPrompt(existingRefFileIDs []string, currentFileID string) string {
	var sb strings.Builder

	// Check if there's history (HISTORY.txt would be in ref_file_ids)
	hasHistory := false
	for _, id := range existingRefFileIDs {
		if strings.TrimSpace(id) != "" && strings.TrimSpace(id) != currentFileID {
			hasHistory = true
			break
		}
	}

	// Strong priority indication for current input
	sb.WriteString("【重要 - 当前任务】\n")
	sb.WriteString("请优先阅读并理解以下文件中的内容，这是你的当前任务：\n")
	sb.WriteString(fmt.Sprintf("[文件引用: %s]\n", currentInputFilename))
	sb.WriteString("\n")

	if hasHistory {
		sb.WriteString("【参考 - 历史上下文】\n")
		sb.WriteString("以下文件包含历史对话上下文，仅供参考，请优先以上述【当前任务】为准：\n")
		sb.WriteString("[文件引用: HISTORY.txt]\n")
		sb.WriteString("\n")
	}

	sb.WriteString("【指令】\n")
	sb.WriteString("请基于【当前任务】文件中的内容回答用户问题。如果当前任务与历史上下文有冲突，请以当前任务为准。")

	return sb.String()
}

// UploadCurrentInputFromRequest uploads the current turn content from the raw request
// before prompt building. This modifies the request to reference the uploaded file.
func UploadCurrentInputFromRequest(ctx context.Context, a *auth.RequestAuth, ds shared.DeepSeekCaller, req map[string]any) (map[string]any, error) {
	if ds == nil || a == nil {
		return req, nil
	}

	messagesRaw, _ := req["messages"].([]any)
	if len(messagesRaw) == 0 {
		return req, nil
	}

	// Build content from all messages (this includes Coding Agent prompt + user input)
	var parts []string
	for _, msg := range messagesRaw {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content := extractMessageContent(m)
		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}
	content := strings.Join(parts, "\n\n")
	if strings.TrimSpace(content) == "" {
		return req, nil
	}

	// Strip internal system prompts
	content = stripInternalSystemPrompts(content)

	// Strip internal markers
	content = stripInternalMarkers(content)

	// Build the transcript
	inputText := buildCurrentInputTranscript(content)
	if strings.TrimSpace(inputText) == "" {
		return req, errors.New("current input split produced empty transcript")
	}

	// Upload the current input as a file
	result, err := ds.UploadFile(ctx, a, dsclient.UploadFileRequest{
		Filename:    currentInputFilename,
		ContentType: currentInputContentType,
		Purpose:     currentInputPurpose,
		Data:        []byte(inputText),
	}, 3)
	if err != nil {
		return req, fmt.Errorf("upload current input file: %w", err)
	}

	fileID := strings.TrimSpace(result.ID)
	if fileID == "" {
		return req, errors.New("upload current input file returned empty file id")
	}

	// Find the last user message index to replace
	_, lastUserIndex := extractLastUserMessage(messagesRaw)
	if lastUserIndex < 0 {
		return req, nil
	}

	// Replace the last user message with a reference to the uploaded file
	replacementMsg := map[string]any{
		"role":    "user",
		"content": fmt.Sprintf("[文件引用: %s]\n请查看上传的文件内容并回答相关问题。", currentInputFilename),
	}

	// Create new messages slice with the replacement
	newMessages := make([]any, len(messagesRaw))
	copy(newMessages, messagesRaw)
	newMessages[lastUserIndex] = replacementMsg

	// Update the request
	req["messages"] = newMessages

	// Add file_id to ref_file_ids if present
	refFileIDsAny, _ := req["ref_file_ids"].([]any)
	refFileIDs := make([]string, 0, len(refFileIDsAny))
	for _, id := range refFileIDsAny {
		if s, ok := id.(string); ok {
			refFileIDs = append(refFileIDs, s)
		}
	}
	refFileIDs = prependUniqueRefFileID(refFileIDs, fileID)
	req["ref_file_ids"] = refFileIDs

	return req, nil
}
