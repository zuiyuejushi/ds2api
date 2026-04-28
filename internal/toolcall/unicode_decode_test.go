package toolcall

import (
	"testing"
)

func TestDecodeUnicodeEscapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no escapes", "hello world", "hello world"},
		{"simple unicode", `\u0048\u0065\u006c\u006c\u006f`, "Hello"},
		{"unicode with braces", `\u{1F600}`, "😀"},
		{"mixed content", `Hello \u2705 World`, "Hello ✅ World"},
		{"chinese characters", `\u4e2d\u6587`, "中文"},
		{"emoji", `\u{1F389}\u{1F38A}`, "🎉🎊"},
		{"incomplete escape", `\u123`, `\u123`},
		{"invalid hex", `\uGGGG`, `\uGGGG`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeUnicodeEscapes(tt.input)
			if got != tt.expected {
				t.Errorf("decodeUnicodeEscapes(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseInvokeParameterValueWithUnicode(t *testing.T) {
	tests := []struct {
		name     string
		inner    string
		expected string
	}{
		{"CDATA with unicode escape", `<![CDATA[Hello \u2705]]>`, "Hello ✅"},
		{"plain text with unicode escape", `Status: \u2705`, "Status: ✅"},
		{"CDATA with chinese", `<![CDATA[\u4e2d\u6587]]>`, "中文"},
		{"mixed unicode and text", `Test \u2713 Check`, "Test ✓ Check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInvokeParameterValue(tt.inner)
			gotStr, ok := got.(string)
			if !ok {
				t.Errorf("expected string, got %T", got)
				return
			}
			if gotStr != tt.expected {
				t.Errorf("parseInvokeParameterValue(%q) = %q, want %q", tt.inner, gotStr, tt.expected)
			}
		})
	}
}
