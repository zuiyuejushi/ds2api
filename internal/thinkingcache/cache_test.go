package thinkingcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAndApplyRestoresMissingAssistantReasoning(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())

	prefix := []any{
		map[string]any{"role": "user", "content": "solve it"},
	}
	Store(prefix, "deepseek-v4-pro", "cached reasoning")

	messages := []any{
		map[string]any{"role": "user", "content": "solve it"},
		map[string]any{"role": "assistant", "content": "answer"},
		map[string]any{"role": "user", "content": "continue"},
	}
	out, changed := Apply(messages, "deepseek-v4-pro")
	if !changed {
		t.Fatal("expected cached reasoning to be injected")
	}
	assistant := out[1].(map[string]any)
	if assistant["reasoning_content"] != "cached reasoning" {
		t.Fatalf("expected cached reasoning_content, got %#v", assistant["reasoning_content"])
	}
}

func TestApplyPrefersCachedReasoningOverExistingReasoningContent(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())

	prefix := []any{
		map[string]any{"role": "user", "content": "solve it"},
	}
	Store(prefix, "deepseek-v4-pro", "cached reasoning")

	messages := []any{
		map[string]any{"role": "user", "content": "solve it"},
		map[string]any{"role": "assistant", "content": "answer", "reasoning_content": "client reasoning"},
	}
	out, changed := Apply(messages, "deepseek-v4-pro")
	if !changed {
		t.Fatal("expected cached reasoning to replace existing reasoning_content")
	}
	assistant := out[1].(map[string]any)
	if assistant["reasoning_content"] != "cached reasoning" {
		t.Fatalf("expected cached reasoning_content, got %#v", assistant["reasoning_content"])
	}
}

func TestApplyNormalizesUnsupportedReasoningField(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())

	messages := []any{
		map[string]any{"role": "user", "content": "question"},
		map[string]any{"role": "assistant", "content": "answer", "reasoning": "client reasoning"},
	}
	out, changed := Apply(messages, "deepseek-v4-pro")
	if !changed {
		t.Fatal("expected unsupported reasoning field to be normalized")
	}
	assistant := out[1].(map[string]any)
	if assistant["reasoning_content"] != "client reasoning" {
		t.Fatalf("expected reasoning_content from reasoning, got %#v", assistant["reasoning_content"])
	}
}

func TestApplyNormalizesContentReasoningBlock(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())

	messages := []any{
		map[string]any{"role": "user", "content": "question"},
		map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{"type": "reasoning", "text": "content reasoning"},
				map[string]any{"type": "text", "text": "answer"},
			},
		},
	}
	out, changed := Apply(messages, "deepseek-v4-pro")
	if !changed {
		t.Fatal("expected content reasoning block to be normalized")
	}
	assistant := out[1].(map[string]any)
	if assistant["reasoning_content"] != "content reasoning" {
		t.Fatalf("expected reasoning_content from content block, got %#v", assistant["reasoning_content"])
	}
}

func TestApplyDoesNotTreatUnknownContentBlockAsReasoning(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())

	messages := []any{
		map[string]any{"role": "user", "content": "question"},
		map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{"type": "unsupported_reasoning", "text": "do not inject"},
				map[string]any{"type": "text", "text": "answer"},
			},
		},
	}
	if _, changed := Apply(messages, "deepseek-v4-pro"); changed {
		t.Fatal("did not expect unknown content block to be normalized as reasoning")
	}
}

func TestApplyIgnoresExpiredEntries(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("DS2API_THINKING_CACHE_DIR", cacheDir)

	prefix := []any{map[string]any{"role": "user", "content": "old"}}
	key := keyFor(prefix, "deepseek-v4-pro")
	if err := os.WriteFile(filepath.Join(cacheDir, key+".dat"), []byte("expired reasoning"), 0o600); err != nil {
		t.Fatalf("write data: %v", err)
	}
	index := map[string]entry{
		key: {
			Size:    len("expired reasoning"),
			Created: time.Now().Add(-3 * time.Hour),
			Expires: time.Now().Add(-time.Minute),
		},
	}
	b, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "index.json"), b, 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}

	messages := []any{
		map[string]any{"role": "user", "content": "old"},
		map[string]any{"role": "assistant", "content": "answer"},
	}
	if _, changed := Apply(messages, "deepseek-v4-pro"); changed {
		t.Fatal("did not expect expired cache entry to be injected")
	}
	if _, err := os.Stat(filepath.Join(cacheDir, key+".dat")); !os.IsNotExist(err) {
		t.Fatalf("expected expired data file removed, err=%v", err)
	}
}

func TestStorePrunesOldestEntriesOverThreshold(t *testing.T) {
	t.Setenv("DS2API_THINKING_CACHE_DIR", t.TempDir())
	t.Setenv("DS2API_THINKING_CACHE_MAX_ENTRIES", "1")

	Store([]any{map[string]any{"role": "user", "content": "one"}}, "deepseek-v4-pro", "first")
	Store([]any{map[string]any{"role": "user", "content": "two"}}, "deepseek-v4-pro", "second")

	index, err := loadIndexLocked()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if len(index) != 1 {
		t.Fatalf("expected one retained entry, got %d", len(index))
	}
}
