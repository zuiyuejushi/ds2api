package toolcall

import (
	"testing"
)

func TestParseInvokeParameterValueBoolean(t *testing.T) {
	tests := []struct {
		name     string
		inner    string
		expected string // Go type: string, bool, or float64
	}{
		{"CDATA true", `<![CDATA[true]]>`, "bool"},
		{"CDATA false", `<![CDATA[false]]>`, "bool"},
		{"plain true", `true`, "bool"},
		{"plain false", `false`, "bool"},
		// Numbers are parsed as float64 (both CDATA and plain)
		{"CDATA 42", `<![CDATA[42]]>`, "float64"},
		{"plain 42", `42`, "float64"},
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
