package history

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ds2api/internal/auth"
	dsclient "ds2api/internal/deepseek/client"
	"ds2api/internal/config"
	"ds2api/internal/httpapi/openai/shared"
	"ds2api/internal/promptcompat"
)

const (
	currentInputFilename    = "INPUT.txt"
	currentInputContentType = "text/plain; charset=utf-8"
	currentInputPurpose     = "assistants"
)

// CurrentInputSplitService handles splitting the current user input into a file
// when it exceeds the model's context limit.
type CurrentInputSplitService struct {
	Store shared.ConfigReader
	DS    shared.DeepSeekCaller
}

// Apply uploads the last user message as a file unconditionally.
// This should be called after history split to ensure we're only processing the current turn.
func (s CurrentInputSplitService) Apply(ctx context.Context, a *auth.RequestAuth, stdReq promptcompat.StandardRequest) (promptcompat.StandardRequest, error) {
	if s.DS == nil || s.Store == nil || a == nil {
		return stdReq, nil
	}

	// Find the last user message
	lastUserMsg, lastUserIndex := extractLastUserMessage(stdReq.Messages)
	if lastUserMsg == nil {
		return stdReq, nil
	}

	// Get the content of the last user message
	content := extractMessageContent(lastUserMsg)
	if strings.TrimSpace(content) == "" {
		return stdReq, nil
	}

	// Always convert current input to file regardless of length
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

	// Log for debugging
	config.Logger.Debug("[current_input_split] uploaded file", "file_id", fileID, "filename", currentInputFilename, "bytes", len(inputText))

	// Replace the last user message with a reference to the uploaded file
	// Use a strong citation format to ensure the model references the file
	replacementMsg := map[string]any{
		"role":    "user",
		"content": fmt.Sprintf("[文件引用: %s]\n请查看上传的文件内容并回答相关问题。", currentInputFilename),
	}

	// Create new messages slice with the replacement
	newMessages := make([]any, len(stdReq.Messages))
	copy(newMessages, stdReq.Messages)
	newMessages[lastUserIndex] = replacementMsg

	// Update the request
	stdReq.Messages = newMessages
	stdReq.RefFileIDs = prependUniqueRefFileID(stdReq.RefFileIDs, fileID)
	stdReq.FinalPrompt, stdReq.ToolNames = promptcompat.BuildOpenAIPrompt(newMessages, stdReq.ToolsRaw, "", stdReq.ToolChoice, stdReq.Thinking)

	// Log final state for debugging
	config.Logger.Debug("[current_input_split] updated request", "ref_file_ids", stdReq.RefFileIDs, "final_prompt_length", len(stdReq.FinalPrompt))

	return stdReq, nil
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
// This uses a prominent format to ensure the model recognizes it as the main input
func buildCurrentInputTranscript(content string) string {
	var sb strings.Builder
	sb.WriteString("[当前用户输入 - Current User Input]\n")
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\n" + strings.Repeat("=", 51) + "\n")
	sb.WriteString("[输入结束 - End of Input]")
	return sb.String()
}
