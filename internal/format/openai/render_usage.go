package openai

import (
	"ds2api/internal/sse"
	"ds2api/internal/util"
)

func BuildChatUsage(finalPrompt, finalThinking, finalText string, refFileIDs []string) map[string]any {
	promptTokens := util.EstimateInputTokens(finalPrompt, refFileIDs)
	reasoningTokens := util.EstimateTokens(finalThinking)
	completionTokens := util.EstimateTokens(finalText)
	return map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": reasoningTokens + completionTokens,
		"total_tokens":      promptTokens + reasoningTokens + completionTokens,
		"completion_tokens_details": map[string]any{
			"reasoning_tokens": reasoningTokens,
		},
	}
}

<<<<<<< Updated upstream
func BuildChatUsageFromUpstream(upstream *sse.TokenUsage, finalPrompt, finalThinking, finalText string) map[string]any {
	if upstream != nil && upstream.PromptTokens > 0 && upstream.CompletionTokens > 0 {
		return map[string]any{
			"prompt_tokens":     upstream.PromptTokens,
			"completion_tokens": upstream.CompletionTokens,
			"total_tokens":      upstream.TotalTokens,
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": util.EstimateTokens(finalThinking),
			},
		}
	}
	return BuildChatUsage(finalPrompt, finalThinking, finalText, nil)
}

func BuildResponsesUsageFromUpstream(upstream *sse.TokenUsage, finalPrompt, finalThinking, finalText string) map[string]any {
	if upstream != nil && upstream.PromptTokens > 0 && upstream.CompletionTokens > 0 {
		return map[string]any{
			"input_tokens":  upstream.PromptTokens,
			"output_tokens": upstream.CompletionTokens,
			"total_tokens":  upstream.TotalTokens,
		}
	}
	return BuildResponsesUsage(finalPrompt, finalThinking, finalText, nil)
}

func BuildResponsesUsage(finalPrompt, finalThinking, finalText string, refFileIDs []string) map[string]any {
	promptTokens := util.EstimateInputTokens(finalPrompt, refFileIDs)
=======
func BuildResponsesUsage(finalPrompt, finalThinking, finalText string, refFileIDs []string) map[string]any {
	promptTokens := util.EstimateInputTokens(finalPrompt, refFileIDs)
>>>>>>> Stashed changes
	reasoningTokens := util.EstimateTokens(finalThinking)
	completionTokens := util.EstimateTokens(finalText)
	return map[string]any{
		"input_tokens":  promptTokens,
		"output_tokens": reasoningTokens + completionTokens,
		"total_tokens":  promptTokens + reasoningTokens + completionTokens,
	}
}
