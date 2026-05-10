package chathistory

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"ds2api/internal/config"
)

// RecordTokenUsage saves token usage record for statistics (retains 30 days)
func (s *Store) RecordTokenUsage(model string, usage map[string]any) {
	if s == nil || usage == nil {
		return
	}

	promptTokens := extractTokenValue(usage, "prompt_tokens", "input_tokens")
	completionTokens := extractTokenValue(usage, "completion_tokens", "output_tokens")
	totalTokens := extractTokenValue(usage, "total_tokens")
	if totalTokens == 0 {
		totalTokens = promptTokens + completionTokens
	}
	cachedTokens := extractTokenValue(usage, "prompt_cache_hit_tokens")

	record := TokenRecord{
		Timestamp:        time.Now().UnixMilli(),
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		CachedTokens:     cachedTokens,
	}

	if err := s.appendTokenRecord(record); err != nil {
		config.Logger.Warn("[token_stats] failed to record", "error", err)
	}
}

// GetTokenStatsPath returns the path for token stats file
func (s *Store) GetTokenStatsPath() string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.path) + ".tokens.json"
}

// appendTokenRecord appends a token record to the stats file
func (s *Store) appendTokenRecord(record TokenRecord) error {
	path := s.GetTokenStatsPath()
	if path == "" {
		return nil
	}

	// Load existing records
	stats := tokenStatsFile{
		Version: tokenStatsVersion,
		Records: []TokenRecord{},
	}

	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &stats)
	}

	// Add new record
	stats.Records = append(stats.Records, record)

	// Clean old records (keep only 30 days)
	cutoff := time.Now().Add(-tokenStatsRetentionDays * 24 * time.Hour).UnixMilli()
	filtered := make([]TokenRecord, 0, len(stats.Records))
	for _, r := range stats.Records {
		if r.Timestamp >= cutoff {
			filtered = append(filtered, r)
		}
	}
	stats.Records = filtered

	// Save back
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// QueryTokenRecords returns token records within the given time range
func (s *Store) QueryTokenRecords(startTime, endTime int64) []TokenRecord {
	if s == nil {
		return nil
	}

	path := s.GetTokenStatsPath()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var stats tokenStatsFile
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil
	}

	var result []TokenRecord
	for _, r := range stats.Records {
		if r.Timestamp >= startTime && r.Timestamp <= endTime {
			result = append(result, r)
		}
	}
	return result
}

func extractTokenValue(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case float64:
				return int64(val)
			case float32:
				return int64(val)
			case int:
				return int64(val)
			case int64:
				return val
			case string:
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return int64(f)
				}
			}
		}
	}
	return 0
}
