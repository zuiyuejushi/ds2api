package util

import (
	"ds2api/internal/claudeconv"
	"ds2api/internal/config"
	"ds2api/internal/filetoken"
	"ds2api/internal/prompt"
	"ds2api/internal/tokenizer"
)

const ClaudeDefaultModel = "claude-sonnet-4-6"

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func MessagesPrepare(messages []map[string]any) string {
	return prompt.MessagesPrepare(messages)
}

func normalizeContent(v any) string {
	return prompt.NormalizeContent(v)
}

func ConvertClaudeToDeepSeek(claudeReq map[string]any, store *config.Store) map[string]any {
	return claudeconv.ConvertClaudeToDeepSeek(claudeReq, store, ClaudeDefaultModel)
}

// EstimateTokens returns an accurate token count using the cl100k_base tokenizer
// (OpenAI tiktoken). Falls back to a character-ratio heuristic if the tokenizer
// is unavailable.
func EstimateTokens(text string) int {
	return tokenizer.CountTokens(text)
}

// EstimateInputTokens returns the total input token count including both the
// rendered prompt text and any attached file content (via ref_file_ids).
// File tokens are looked up from the filetoken cache populated during upload.
func EstimateInputTokens(finalPrompt string, refFileIDs []string) int {
	tokens := EstimateTokens(finalPrompt)
	for _, id := range refFileIDs {
		tokens += filetoken.Get(id)
	}
	return tokens
}
