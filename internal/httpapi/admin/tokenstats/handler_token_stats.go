package tokenstats

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ds2api/internal/chathistory"
)

type TimeRange string

const (
	Range6Hours  TimeRange = "6h"
	Range24Hours TimeRange = "24h"
	Range7Days   TimeRange = "7d"
	Range30Days  TimeRange = "30d"
)

// DeepSeek official pricing (per 1M tokens)
// Reference: https://platform.deepseek.com/pricing
var modelPricing = map[string]struct {
	InputPrice  float64 // $ per 1M input tokens
	OutputPrice float64 // $ per 1M output tokens
}{
	// DeepSeek V4 Flash series (default/cheaper)
	"deepseek-v4-flash":         {InputPrice: 0.50, OutputPrice: 1.00},
	"deepseek-v4-flash-search":  {InputPrice: 0.50, OutputPrice: 1.00},
	"deepseek-v4-vision":        {InputPrice: 0.50, OutputPrice: 1.00},
	"deepseek-v4-vision-search": {InputPrice: 0.50, OutputPrice: 1.00},
	// DeepSeek V4 Pro series (more expensive, reasoning model)
	"deepseek-v4-pro":        {InputPrice: 2.00, OutputPrice: 6.00},
	"deepseek-v4-pro-search": {InputPrice: 2.00, OutputPrice: 6.00},
}

// Default pricing for unknown models (use Flash pricing as default)
var defaultPricing = struct {
	InputPrice  float64
	OutputPrice float64
}{InputPrice: 0.50, OutputPrice: 1.00}

type StatsPoint struct {
	Timestamp int64 `json:"timestamp"`
	// Token counts
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
	// Request counts
	RequestCount int64 `json:"request_count"`
	// Cost (calculated based on actual DeepSeek pricing)
	Cost float64 `json:"cost"`
}

type TokenStatsResponse struct {
	// Summary stats
	TotalRequests     int64   `json:"total_requests"`
	TotalCost         float64 `json:"total_cost"`
	TotalPromptTokens int64   `json:"total_prompt_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalTokens       int64   `json:"total_tokens"`
	CachedTokens      int64   `json:"cached_tokens"`
	// Breakdown by model
	ModelBreakdown map[string]ModelStats `json:"model_breakdown"`
	// Time series data
	Points []StatsPoint `json:"points"`
	// Metadata
	Range     string `json:"range"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type ModelStats struct {
	Requests         int64   `json:"requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

func (h *Handler) GetTokenStats(w http.ResponseWriter, r *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}

	// Parse time range parameter
	rangeParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("range")))
	timeRange := parseTimeRange(rangeParam)

	// Calculate time boundaries
	now := time.Now()
	var startTime time.Time
	var interval time.Duration
	var pointCount int

	switch timeRange {
	case Range6Hours:
		startTime = now.Add(-6 * time.Hour)
		interval = time.Hour
		pointCount = 6
	case Range24Hours:
		startTime = now.Add(-24 * time.Hour)
		interval = time.Hour
		pointCount = 24
	case Range7Days:
		startTime = now.Add(-7 * 24 * time.Hour)
		interval = 24 * time.Hour
		pointCount = 7
	case Range30Days:
		startTime = now.Add(-30 * 24 * time.Hour)
		interval = 24 * time.Hour
		pointCount = 30
	default:
		startTime = now.Add(-24 * time.Hour)
		interval = time.Hour
		pointCount = 24
	}

	// Query token records from dedicated token stats store
	records := store.QueryTokenRecords(startTime.UnixMilli(), now.UnixMilli())

	// Initialize points
	points := make([]StatsPoint, pointCount)
	for i := 0; i < pointCount; i++ {
		pointTime := startTime.Add(time.Duration(i) * interval)
		points[i] = StatsPoint{Timestamp: pointTime.UnixMilli()}
	}

	// Aggregate data
	var totalRequests int64
	var totalPromptTokens int64
	var totalCompletionTokens int64
	var totalTokens int64
	var cachedTokens int64
	var totalCost float64
	modelBreakdown := make(map[string]ModelStats)

	for _, record := range records {
		// Get model and pricing
		model := record.Model
		if model == "" {
			model = "unknown"
		}
		pricing := getPricing(model)

		// Calculate cost for this request
		requestCost := calculateCost(record.PromptTokens, record.CompletionTokens, pricing)

		// Update totals
		totalRequests++
		totalPromptTokens += record.PromptTokens
		totalCompletionTokens += record.CompletionTokens
		totalTokens += record.TotalTokens
		totalCost += requestCost
		cachedTokens += record.CachedTokens

		// Update model breakdown
		modelStat := modelBreakdown[model]
		modelStat.Requests++
		modelStat.PromptTokens += record.PromptTokens
		modelStat.CompletionTokens += record.CompletionTokens
		modelStat.TotalTokens += record.TotalTokens
		modelStat.Cost += requestCost
		modelBreakdown[model] = modelStat

		// Find which point this belongs to
		pointIndex := int(record.Timestamp-startTime.UnixMilli()) / int(interval.Milliseconds())
		if pointIndex >= 0 && pointIndex < pointCount {
			points[pointIndex].PromptTokens += record.PromptTokens
			points[pointIndex].CompletionTokens += record.CompletionTokens
			points[pointIndex].TotalTokens += record.TotalTokens
			points[pointIndex].RequestCount++
			points[pointIndex].Cost += requestCost
		}
	}

	response := TokenStatsResponse{
		TotalRequests:     totalRequests,
		TotalCost:         totalCost,
		TotalPromptTokens: totalPromptTokens,
		TotalOutputTokens: totalCompletionTokens,
		TotalTokens:       totalTokens,
		CachedTokens:      cachedTokens,
		ModelBreakdown:    modelBreakdown,
		Points:            points,
		Range:             string(timeRange),
		StartTime:         startTime.UnixMilli(),
		EndTime:           now.UnixMilli(),
	}

	writeJSON(w, http.StatusOK, response)
}

// RecordTokenUsage records token usage for a completed request
func RecordTokenUsage(store *chathistory.Store, id, model string, usage map[string]any) error {
	if store == nil {
		return nil
	}
	store.RecordTokenUsage(model, usage)
	return nil
}

// getPricing returns the pricing for a given model
// Supports model aliases and falls back to default pricing
func getPricing(model string) struct{ InputPrice, OutputPrice float64 } {
	if model == "" {
		return defaultPricing
	}

	// Try exact match first
	if pricing, ok := modelPricing[model]; ok {
		return pricing
	}

	// Try to match by model family
	modelLower := strings.ToLower(model)

	// Check for Pro models (more expensive)
	if strings.Contains(modelLower, "pro") || strings.Contains(modelLower, "opus") {
		if pricing, ok := modelPricing["deepseek-v4-pro"]; ok {
			return pricing
		}
	}

	// Check for Flash models (cheaper) - default
	if pricing, ok := modelPricing["deepseek-v4-flash"]; ok {
		return pricing
	}

	return defaultPricing
}

// calculateCost calculates the cost based on token counts and pricing
func calculateCost(promptTokens, completionTokens int64, pricing struct{ InputPrice, OutputPrice float64 }) float64 {
	inputCost := float64(promptTokens) / 1_000_000 * pricing.InputPrice
	outputCost := float64(completionTokens) / 1_000_000 * pricing.OutputPrice
	return inputCost + outputCost
}

func parseTimeRange(s string) TimeRange {
	switch s {
	case "6h":
		return Range6Hours
	case "24h":
		return Range24Hours
	case "7d":
		return Range7Days
	case "30d":
		return Range30Days
	default:
		return Range30Days
	}
}

func extractInt64(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return int64(toFloat64(v))
		}
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		return 0
	default:
		return 0
	}
}
