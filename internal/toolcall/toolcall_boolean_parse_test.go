package toolcall

import (
	"testing"
)

func TestParseInvokeParameterValueBoolean(t *testing.T) {
	tests := []struct {
		name     string
		inner    string
		expected string // Go type: string or bool
	}{
		{"CDATA true", `<![CDATA[true]]>`, "bool"},
		{"CDATA false", `<![CDATA[false]]>`, "bool"},
		{"plain true", `true`, "bool"},
		{"plain false", `false`, "bool"},
		// Numbers are intentionally kept as strings to preserve values like "1"
		{"CDATA 42", `<![CDATA[42]]>`, "string"},
		{"plain 42", `42`, "string"},
		{"CDATA string", `<![CDATA[hello]]>`, "string"},
		{"plain string", `hello`, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInvokeParameterValue(tt.inner)
			actualType := "string"
			switch got.(type) {
			case bool:
				actualType = "bool"
			case float64:
				actualType = "float64"
			}
			if actualType != tt.expected {
				t.Errorf("%s: got %T(%v), want type %s", tt.name, got, got, tt.expected)
			}
		})
	}
}
