package toolcall

import (
	"testing"
)

func TestForceLimitTo2000(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected int
	}{
		{
			name: "limit should be forced to 2000",
			xml: `<tool_calls>
				<invoke name="read">
					<parameter name="file_path">test.go</parameter>
					<parameter name="limit">100</parameter>
				</invoke>
			</tool_calls>`,
			expected: 2000,
		},
		{
			name: "limit 50 should become 2000",
			xml: `<tool_calls>
				<invoke name="read">
					<parameter name="file_path">test.go</parameter>
					<parameter name="limit">50</parameter>
				</invoke>
			</tool_calls>`,
			expected: 2000,
		},
		{
			name: "JSON format limit should be forced to 2000",
			xml: `<tool_calls>
				<invoke name="read">
					{"input": {"file_path": "test.go", "limit": 100}}
				</invoke>
			</tool_calls>`,
			expected: 2000,
		},
		{
			name: "no limit should not add limit",
			xml: `<tool_calls>
				<invoke name="read">
					<parameter name="file_path">test.go</parameter>
				</invoke>
			</tool_calls>`,
			expected: -1, // -1 means no limit key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := ParseToolCalls(tt.xml, []string{"read"})
			if len(calls) == 0 {
				t.Fatal("expected at least one tool call")
			}

			input := calls[0].Input
			limitVal, ok := input["limit"]

			if tt.expected == -1 {
				if ok {
					t.Errorf("expected no limit, but got %v", limitVal)
				}
				return
			}

			if !ok {
				t.Fatal("expected limit to be present")
			}

			limit, ok := limitVal.(int)
			if !ok {
				t.Fatalf("expected limit to be int, got %T", limitVal)
			}

			if limit != tt.expected {
				t.Errorf("expected limit=%d, got limit=%d", tt.expected, limit)
			}
		})
	}
}
