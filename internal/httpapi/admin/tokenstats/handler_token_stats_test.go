package tokenstats

import (
	"math"
	"testing"
)

func TestExtractInt64(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		keys     []string
		expected int64
	}{
		{
			name:     "extract prompt_tokens",
			m:        map[string]any{"prompt_tokens": 100},
			keys:     []string{"prompt_tokens", "input_tokens"},
			expected: 100,
		},
		{
			name:     "extract input_tokens fallback",
			m:        map[string]any{"input_tokens": 200},
			keys:     []string{"prompt_tokens", "input_tokens"},
			expected: 200,
		},
		{
			name:     "extract float64",
			m:        map[string]any{"prompt_tokens": 150.5},
			keys:     []string{"prompt_tokens"},
			expected: 150,
		},
		{
			name:     "extract int",
			m:        map[string]any{"prompt_tokens": 300},
			keys:     []string{"prompt_tokens"},
			expected: 300,
		},
		{
			name:     "key not found",
			m:        map[string]any{"other": 100},
			keys:     []string{"prompt_tokens"},
			expected: 0,
		},
		{
			name:     "extract string number",
			m:        map[string]any{"prompt_tokens": "500"},
			keys:     []string{"prompt_tokens"},
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInt64(tt.m, tt.keys...)
			if result != tt.expected {
				t.Errorf("extractInt64() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		v        any
		expected float64
	}{
		{"float64", 123.45, 123.45},
		{"float32", float32(123.45), float64(float32(123.45))},
		{"int", 100, 100},
		{"int64", int64(200), 200},
		{"int32", int32(300), 300},
		{"string number", "500", 500},
		{"string invalid", "abc", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.v)
			if result != tt.expected {
				t.Errorf("toFloat64() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		input    string
		expected TimeRange
	}{
		{"30s", Range30Seconds},
		{"24h", Range24Hours},
		{"7d", Range7Days},
		{"30d", Range30Days},
		{"", Range30Days},
		{"invalid", Range30Days},
		{"30S", Range30Days},
		{"24H", Range30Days},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTimeRange(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeRange(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetPricing(t *testing.T) {
	tests := []struct {
		model                string
		expectedInputPrice   float64
		expectedOutputPrice  float64
	}{
		{"deepseek-v4-flash", 0.50, 1.00},
		{"deepseek-v4-flash-search", 0.50, 1.00},
		{"deepseek-v4-pro", 2.00, 6.00},
		{"deepseek-v4-pro-search", 2.00, 6.00},
		{"deepseek-v4-vision", 0.50, 1.00},
		{"unknown-model", 0.50, 1.00}, // default to flash pricing
		{"", 0.50, 1.00},              // empty model defaults to flash
		{"claude-opus-4-6", 2.00, 6.00}, // contains "opus", should use pro pricing
		{"some-pro-model", 2.00, 6.00},  // contains "pro", should use pro pricing
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing := getPricing(tt.model)
			if pricing.InputPrice != tt.expectedInputPrice {
				t.Errorf("getPricing(%q).InputPrice = %v, want %v", tt.model, pricing.InputPrice, tt.expectedInputPrice)
			}
			if pricing.OutputPrice != tt.expectedOutputPrice {
				t.Errorf("getPricing(%q).OutputPrice = %v, want %v", tt.model, pricing.OutputPrice, tt.expectedOutputPrice)
			}
		})
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name          string
		promptTokens  int64
		completionTokens int64
		pricing       struct{ InputPrice, OutputPrice float64 }
		expected      float64
	}{
		{
			name:             "flash model - 1M tokens each",
			promptTokens:     1_000_000,
			completionTokens: 1_000_000,
			pricing:          struct{ InputPrice, OutputPrice float64 }{InputPrice: 0.50, OutputPrice: 1.00},
			expected:         1.50, // $0.50 + $1.00
		},
		{
			name:             "pro model - 1M tokens each",
			promptTokens:     1_000_000,
			completionTokens: 1_000_000,
			pricing:          struct{ InputPrice, OutputPrice float64 }{InputPrice: 2.00, OutputPrice: 6.00},
			expected:         8.00, // $2.00 + $6.00
		},
		{
			name:             "flash model - 500K input, 200K output",
			promptTokens:     500_000,
			completionTokens: 200_000,
			pricing:          struct{ InputPrice, OutputPrice float64 }{InputPrice: 0.50, OutputPrice: 1.00},
			expected:         0.45, // $0.25 + $0.20
		},
		{
			name:             "zero tokens",
			promptTokens:     0,
			completionTokens: 0,
			pricing:          struct{ InputPrice, OutputPrice float64 }{InputPrice: 0.50, OutputPrice: 1.00},
			expected:         0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCost(tt.promptTokens, tt.completionTokens, tt.pricing)
			// Use small epsilon for float comparison
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("calculateCost() = %v, want %v", result, tt.expected)
			}
		})
	}
}
