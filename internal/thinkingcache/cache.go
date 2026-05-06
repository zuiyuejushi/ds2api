package thinkingcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ds2api/internal/config"
)

const (
	defaultDirRel     = "data/thinking_cache"
	defaultMaxEntries = 500
	ttl               = 120 * time.Minute
)

var mu sync.Mutex

type entry struct {
	Thinking  string    `json:"thinking,omitempty"`
	Signature string    `json:"signature,omitempty"`
	Size      int       `json:"size,omitempty"`
	Created   time.Time `json:"created"`
	Expires   time.Time `json:"expires"`
}

type indexedEntry struct {
	Key   string
	Meta  entry
	Order time.Time
}

// Apply restores assistant reasoning into historical messages before prompt
// rendering. Cached reasoning takes priority over existing reasoning fields.
func Apply(messages []any, model string) ([]any, bool) {
	if len(messages) == 0 {
		return messages, false
	}
	mu.Lock()
	defer mu.Unlock()

	index, _ := loadIndexLocked()
	index = pruneExpiredLocked(index, time.Now())

	var out []any
	changed := false
	cacheRestored := 0

	for i, item := range messages {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(asString(msg["role"])))
		if role != "assistant" {
			continue
		}

		key := keyFor(messages[:i], model)
		if key == "" {
			continue
		}

		cached := readThinkingLocked(index, key)
		if strings.TrimSpace(cached) == "" {
			continue
		}

		// Cache takes priority: use cached thinking regardless of existing content
		existing := strings.TrimSpace(asString(msg["reasoning_content"]))
		if existing == cached {
			continue
		}

		if out == nil {
			out = append([]any(nil), messages...)
		}
		cloned := cloneMap(msg)
		cloned["reasoning_content"] = cached
		out[i] = cloned
		cacheRestored++
		changed = true
	}

	if !changed {
		return messages, false
	}

	config.Logger.Info(
		"[thinking_cache] injected assistant reasoning",
		"model", strings.TrimSpace(model),
		"cache_restored", cacheRestored,
	)
	return out, true
}

// Store saves the current assistant turn's reasoning under the fingerprint of
// the prompt-visible messages that preceded it.
func Store(messages []any, model, thinking string) {
	thinking = strings.TrimSpace(thinking)
	if len(messages) == 0 || thinking == "" {
		return
	}

	key := keyFor(messages, model)
	if key == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	index, _ := loadIndexLocked()
	index = pruneExpiredLocked(index, now)

	if err := os.MkdirAll(dir(), 0o755); err != nil {
		config.Logger.Warn("[thinking_cache] mkdir failed", "error", err)
		return
	}

	if err := os.WriteFile(dataPath(key), []byte(thinking), 0o600); err != nil {
		config.Logger.Warn("[thinking_cache] write data failed", "key", key, "error", err)
		return
	}

	index[key] = entry{
		Thinking:  "",
		Signature: "",
		Size:      len([]byte(thinking)),
		Created:   now,
		Expires:   now.Add(ttl),
	}

	index = pruneOverflowLocked(index, maxEntries())

	if err := saveIndexLocked(index); err != nil {
		config.Logger.Warn("[thinking_cache] write index failed", "error", err)
		return
	}

	config.Logger.Info(
		"[thinking_cache] stored assistant reasoning",
		"model", strings.TrimSpace(model),
		"key", key,
		"size", len([]byte(thinking)),
	)
}

func hasThinkingBlock(msg map[string]any) bool {
	content, ok := msg["content"].([]any)
	if !ok {
		return false
	}
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		blockType := strings.ToLower(strings.TrimSpace(asString(block["type"])))
		if blockType == "thinking" || blockType == "redacted_thinking" {
			return true
		}
	}
	return false
}

func readThinkingLocked(index map[string]entry, key string) string {
	meta, ok := index[key]
	if !ok || time.Now().After(meta.Expires) {
		return ""
	}
	if text := strings.TrimSpace(meta.Thinking); text != "" {
		return text
	}
	b, err := os.ReadFile(dataPath(key))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func firstReasoning(msg map[string]any) (string, string) {
	for _, key := range []string{"reasoning_content", "reasoning", "thinking", "thinking_content"} {
		if text := strings.TrimSpace(reasoningString(msg[key])); text != "" {
			return text, key
		}
	}
	if text := strings.TrimSpace(contentReasoning(msg["content"])); text != "" {
		return text, "content_reasoning"
	}
	return "", ""
}

func reasoningString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		for _, key := range []string{"text", "content", "thinking", "reasoning"} {
			if text := strings.TrimSpace(asString(x[key])); text != "" {
				return text
			}
		}
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if text := strings.TrimSpace(reasoningString(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func contentReasoning(v any) string {
	items, ok := v.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(asString(m["type"]))) {
		case "reasoning", "thinking":
			if text := strings.TrimSpace(reasoningString(m)); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func keyFor(messages []any, model string) string {
	normalized := map[string]any{
		"model":    strings.TrimSpace(model),
		"messages": normalizeForFingerprint(messages),
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16]
}

func normalizeForFingerprint(messages []any) []any {
	out := make([]any, 0, len(messages))
	for _, item := range messages {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}

		role := strings.ToLower(strings.TrimSpace(asString(msg["role"])))
		if role == "system" {
			continue
		}

		normalized := make(map[string]any)
		keys := make([]string, 0, len(msg))

		for k := range msg {
			lower := strings.ToLower(strings.TrimSpace(k))
			switch lower {
			case "reasoning", "reasoning_content", "thinking", "thinking_content":
				continue
			}
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for _, k := range keys {
			if k == "content" {
				normalized[k] = normalizeContentForFingerprint(msg[k])
				continue
			}
			normalized[k] = msg[k]
		}
		out = append(out, normalized)
	}
	return out
}

func normalizeContentForFingerprint(v any) any {
	items, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}

		blockType := strings.ToLower(strings.TrimSpace(asString(m["type"])))
		if blockType == "reasoning" || blockType == "thinking" {
			continue
		}

		// Normalize block structure
		normalized := make(map[string]any)
		switch blockType {
		case "text":
			normalized["type"] = "text"
			normalized["text"] = m["text"]
		case "tool_use":
			normalized["type"] = "tool_use"
			normalized["id"] = m["id"]
			normalized["name"] = m["name"]
			normalized["input"] = m["input"]
		case "tool_result":
			normalized["type"] = "tool_result"
			normalized["tool_use_id"] = m["tool_use_id"]
			normalized["content"] = m["content"]
		default:
			normalized["type"] = blockType
		}
		out = append(out, normalized)
	}
	return out
}

func loadIndexLocked() (map[string]entry, error) {
	b, err := os.ReadFile(indexPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]entry{}, nil
		}
		return map[string]entry{}, err
	}
	var index map[string]entry
	if err := json.Unmarshal(b, &index); err != nil {
		return map[string]entry{}, err
	}
	if index == nil {
		index = map[string]entry{}
	}
	return index, nil
}

func saveIndexLocked(index map[string]entry) error {
	if err := os.MkdirAll(dir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath(), append(b, '\n'), 0o600)
}

func pruneExpiredLocked(index map[string]entry, now time.Time) map[string]entry {
	for key, meta := range index {
		if meta.Expires.IsZero() || now.Before(meta.Expires) {
			continue
		}
		delete(index, key)
		removeDataLocked(key)
	}
	return index
}

func pruneOverflowLocked(index map[string]entry, limit int) map[string]entry {
	if limit <= 0 || len(index) <= limit {
		return index
	}
	items := make([]indexedEntry, 0, len(index))
	for key, meta := range index {
		order := meta.Created
		if order.IsZero() {
			order = meta.Expires
		}
		items = append(items, indexedEntry{Key: key, Meta: meta, Order: order})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Order.Before(items[j].Order)
	})
	for len(items) > limit {
		victim := items[0]
		items = items[1:]
		delete(index, victim.Key)
		removeDataLocked(victim.Key)
	}
	return index
}

func removeDataLocked(key string) {
	if err := os.Remove(dataPath(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		config.Logger.Warn("[thinking_cache] remove data failed", "key", key, "error", err)
	}
}

func dir() string {
	return config.ResolvePath("DS2API_THINKING_CACHE_DIR", defaultDirRel)
}

func indexPath() string {
	return filepath.Join(dir(), "index.json")
}

func dataPath(key string) string {
	return filepath.Join(dir(), fmt.Sprintf("%s.dat", key))
}

func maxEntries() int {
	raw := strings.TrimSpace(os.Getenv("DS2API_THINKING_CACHE_MAX_ENTRIES"))
	if raw == "" {
		return defaultMaxEntries
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultMaxEntries
	}
	return n
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
