package toolcall

import (
	"testing"
)

func TestSimplifyReadToolInput(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		xml          string
		expectedKeys []string
	}{
		{
			name:     "Read tool should keep only first parameter",
			toolName: "read",
			xml: `<tool_calls>
				<invoke name="read">
					<parameter name="file_path">test.go</parameter>
					<parameter name="limit">100</parameter>
					<parameter name="offset">50</parameter>
				</invoke>
			</tool_calls>`,
			expectedKeys: []string{"file_path"},
		},
		{
			name:     "Read tool JSON format should keep only first parameter",
			toolName: "read",
			xml: `<tool_calls>
				<invoke name="read">
					{"input": {"file_path": "test.go", "limit": 100, "offset": 50}}
				</invoke>
			</tool_calls>`,
			expectedKeys: []string{"file_path"},
		},
		{
			name:     "Non-read tool should keep all parameters",
			toolName: "bash",
			xml: `<tool_calls>
				<invoke name="bash">
					<parameter name="command">echo hello</parameter>
					<parameter name="timeout">30</parameter>
				</invoke>
			</tool_calls>`,
			expectedKeys: []string{"command", "timeout"},
		},
		{
			name:     "Read tool with only path parameter",
			toolName: "read",
			xml: `<tool_calls>
				<invoke name="read">
					<parameter name="file_path">/path/to/file.txt</parameter>
				</invoke>
			</tool_calls>`,
			expectedKeys: []string{"file_path"},
		},
		{
			name:     "FileRead tool should also be simplified",
			toolName: "FileRead",
			xml: `<tool_calls>
				<invoke name="FileRead">
					<parameter name="path">/path/to/file.txt</parameter>
					<parameter name="offset">0</parameter>
					<parameter name="limit">100</parameter>
				</invoke>
			</tool_calls>`,
			expectedKeys: []string{"path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := ParseToolCalls(tt.xml, []string{tt.toolName})
			if len(calls) == 0 {
				t.Fatal("expected at least one tool call")
			}

			input := calls[0].Input

			if len(input) != len(tt.expectedKeys) {
				t.Errorf("expected %d keys %v, got %d keys %v", len(tt.expectedKeys), tt.expectedKeys, len(input), getKeys(input))
			}

			for _, key := range tt.expectedKeys {
				if _, ok := input[key]; !ok {
					t.Errorf("expected key %q not found in input", key)
				}
			}
		})
	}
}

func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestIsReadTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{"read", "read", true},
		{"Read", "Read", true},
		{"READ", "READ", true},
		{"FileRead", "FileRead", true},
		{"file_read", "file_read", true},
		{"read_file", "read_file", true},
		{"bash", "bash", false},
		{"write", "write", false},
		{"search", "search", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReadTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("isReadTool(%q) = %v, expected %v", tt.toolName, result, tt.expected)
			}
		})
	}
}
