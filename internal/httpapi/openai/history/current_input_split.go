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
	contextFilename         = "CONTEXT.txt"
	currentInputContentType = "text/plain; charset=utf-8"
	currentInputPurpose     = "assistants"
)

// Internal system prompt content that should be excluded
var internalSystemPromptPatterns = []string{
	"Continue the conversation from the full prior context and the latest tool results.",
	"Treat earlier messages as binding context; answer the user's current request as a continuation, not a restart.",
	"Keep reasoning internal. Do not leave the final user-facing answer only in reasoning; always provide the answer in visible assistant content.",
	"You have access to these tools:",
	"TOOL CALL FORMAT",
	"Remember: The ONLY valid way to use tools is the <tool_calls>",
	"JSON FORMAT IS STRICTLY FORBIDDEN",
	"BEFORE FINALIZING YOUR TOOL CALL OUTPUT",
	"Now output THE CORRECT TOOL CALL BLOCK AND NOTHING ELSE",
}

// Internal prompt markers that should be stripped
var internalPromptMarkers = []*regexp.Regexp{
	regexp.MustCompile(`<｜begin▁of▁sentence｜>`),
	regexp.MustCompile(`<｜System｜>`),
	regexp.MustCompile(`<｜User｜>`),
	regexp.MustCompile(`<｜Assistant｜>`),
	regexp.MustCompile(`<｜Tool｜>`),
	regexp.MustCompile(`<｜end▁of▁sentence｜>`),
	regexp.MustCompile(`<｜end▁of▁toolresults｜>`),
	regexp.MustCompile(`<｜end▁of▁instructions｜>`),
	regexp.MustCompile(`\d{13}`),
}

// CurrentInputSplitService handles splitting the current user input into a file
type CurrentInputSplitService struct {
	Store shared.ConfigReader
	DS    shared.DeepSeekCaller
}

// Apply uploads history + current input as a single file unconditionally.
// History is placed first, current input is placed last (closest to user question).
// This excludes our internal system prompts.
func (s CurrentInputSplitService) Apply(ctx context.Context, a *auth.RequestAuth, stdReq promptcompat.StandardRequest) (promptcompat.StandardRequest, error) {
	if s.DS == nil || s.Store == nil || a == nil {
		return stdReq, nil
	}

	// Get history text if available (keep full content for the file)
	historyText := strings.TrimSpace(stdReq.HistoryText)

	// Build current turn content from messages (keep full content for the file)
	currentContent := buildCurrentTurnContent(stdReq.Messages)

	// Combine history + current input into one file
	// History first, current input last (for priority)
	combinedContent := buildCombinedTranscript(historyText, currentContent)
	if strings.TrimSpace(combinedContent) == "" {
		return stdReq, nil
	}

	// Upload as a single file
	result, err := s.DS.UploadFile(ctx, a, dsclient.UploadFileRequest{
		Filename:    contextFilename,
		ContentType: currentInputContentType,
		Purpose:     currentInputPurpose,
		Data:        []byte(combinedContent),
	}, 3)
	if err != nil {
		return stdReq, fmt.Errorf("upload context file: %w", err)
	}

	fileID := strings.TrimSpace(result.ID)
	if fileID == "" {
		return stdReq, errors.New("upload context file returned empty file id")
	}

	// Find the last user message index to replace
	_, lastUserIndex := extractLastUserMessage(stdReq.Messages)
	if lastUserIndex < 0 {
		return stdReq, nil
	}

	// Build replacement message with file reference
	replacementContent := buildContextPrompt()

	replacementMsg := map[string]any{
		"role":    "user",
		"content": replacementContent,
	}

	// Create new messages slice, filtering out internal system prompts and replacing last user message
	newMessages := make([]any, 0, len(stdReq.Messages))
	for i, msg := range stdReq.Messages {
		if i == lastUserIndex {
			// Replace the last user message with file reference
			newMessages = append(newMessages, replacementMsg)
			continue
		}
		// Filter out system messages containing internal prompts
		if isInternalSystemMessage(msg) {
			continue
		}
		newMessages = append(newMessages, msg)
	}

	// Update the request - clear history text since it's now in the file
	stdReq.Messages = newMessages
	stdReq.HistoryText = ""
	stdReq.RefFileIDs = []string{fileID}
	stdReq.FinalPrompt, stdReq.ToolNames = promptcompat.BuildOpenAIPrompt(newMessages, stdReq.ToolsRaw, "", stdReq.ToolChoice, stdReq.Thinking)

	return stdReq, nil
}

// buildCombinedTranscript builds a single transcript with history first, current input last
func buildCombinedTranscript(historyText, currentContent string) string {
	var sb strings.Builder

	// History section (if exists) - placed first (lower priority)
	if historyText != "" {
		sb.WriteString(historyText)
		sb.WriteString("\n\n")
	}

	// Current input section - placed last (highest priority, closest to user)
	if currentContent != "" {
		sb.WriteString(currentContent)
	}

	return sb.String()
}

// buildContextPrompt builds the prompt that references the combined context file
func buildContextPrompt() string {
	var sb strings.Builder
	sb.WriteString("[file content end]\n\n")
	sb.WriteString(fmt.Sprintf("[file name]: %s\n", contextFilename))
	sb.WriteString("[file content begin]\n")
	return sb.String()
}

// buildCurrentTurnContent builds content from all current turn messages
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

// isInternalSystemMessage checks if a message is a system message containing internal prompts
func isInternalSystemMessage(msg any) bool {
	m, ok := msg.(map[string]any)
	if !ok {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(shared.AsString(m["role"])))
	if role != "system" {
		return false
	}
	content := extractMessageContent(m)
	// Check if content contains any internal system prompt patterns
	for _, pattern := range internalSystemPromptPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
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

	// Build content from all messages
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
		Filename:    contextFilename,
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
		"content": fmt.Sprintf("[文件引用: %s]\n请查看上传的文件内容并回答相关问题。", contextFilename),
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
