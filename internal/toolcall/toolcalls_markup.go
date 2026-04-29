package toolcall

import (
	"encoding/json"
	"regexp"
	"strings"
)

var toolCallMarkupKVPattern = regexp.MustCompile(`(?is)<(?:[a-z0-9_:-]+:)?([a-z0-9_\-.]+)\b[^>]*>(.*?)</(?:[a-z0-9_:-]+:)?([a-z0-9_\-.]+)>`)

// cdataPattern matches a standalone CDATA section.
var cdataPattern = regexp.MustCompile(`(?is)^<!\[CDATA\[(.*?)]]>$`)

// unicodeEscapePattern matches JSON-style Unicode escape sequences like \uXXXX or \u{XXXXXX}
var unicodeEscapePattern = regexp.MustCompile(`\\u([0-9a-fA-F]{4})|\\u\{([0-9a-fA-F]+)\}`)

// decodeUnicodeEscapes decodes JSON-style \uXXXX and \u{XXXXXX} Unicode escape sequences.
// This handles cases where models output Unicode escapes in XML parameters that won't be
// automatically decoded by standard HTML/XML unescaping.
// DISABLED: Unicode decoding is disabled.
func decodeUnicodeEscapes(s string) string {
	// DISABLED: Return original string without decoding
	return s
	/*
	return unicodeEscapePattern.ReplaceAllStringFunc(s, func(match string) string {
		submatches := unicodeEscapePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		// Check which group matched (\uXXXX or \u{XXXXXX})
		hexStr := submatches[1]
		if hexStr == "" && len(submatches) > 2 {
			hexStr = submatches[2]
		}
		if hexStr == "" {
			return match
		}
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(codePoint))
	})
	*/
}

func parseMarkupKVObject(text string) map[string]any {
	matches := toolCallMarkupKVPattern.FindAllStringSubmatch(strings.TrimSpace(text), -1)
	if len(matches) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		key := strings.TrimSpace(m[1])
		endKey := strings.TrimSpace(m[3])
		if key == "" {
			continue
		}
		if !strings.EqualFold(key, endKey) {
			continue
		}
		value := parseMarkupValue(m[2])
		if value == nil {
			continue
		}
		appendMarkupValue(out, key, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseMarkupValue(inner string) any {
	if value, ok := extractStandaloneCDATA(inner); ok {
		return value
	}
	value := strings.TrimSpace(extractRawTagValue(inner))
	if value == "" {
		return ""
	}

	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		if parsed := parseStructuredToolCallInput(value); len(parsed) > 0 {
			if len(parsed) == 1 {
				if raw, ok := parsed["_raw"].(string); ok {
					return raw
				}
			}
			return parsed
		}
	}

	var jsonValue any
	if json.Unmarshal([]byte(value), &jsonValue) == nil {
		return jsonValue
	}
	return value
}

func appendMarkupValue(out map[string]any, key string, value any) {
	if existing, ok := out[key]; ok {
		switch current := existing.(type) {
		case []any:
			out[key] = append(current, value)
		default:
			out[key] = []any{current, value}
		}
		return
	}
	out[key] = value
}

// extractRawTagValue treats the inner content of a tag robustly.
// It detects CDATA and strips it, otherwise it unescapes standard HTML entities.
// It avoids over-aggressive tag stripping that might break user content.
// NOTE: Unicode decoding is disabled, but CDATA extraction is kept for boolean parsing.
func extractRawTagValue(inner string) string {
	trimmed := strings.TrimSpace(inner)
	if trimmed == "" {
		return ""
	}

	// 1. Check for CDATA - if present, extract content but skip Unicode decoding
	if value, ok := extractStandaloneCDATA(trimmed); ok {
		return value // Return raw content without Unicode decoding
	}

	// 2. No CDATA, return trimmed value without Unicode decoding
	return trimmed
}

// extractStandaloneCDATA extracts content from CDATA section.
func extractStandaloneCDATA(inner string) (string, bool) {
	trimmed := strings.TrimSpace(inner)
	if cdataMatches := cdataPattern.FindStringSubmatch(trimmed); len(cdataMatches) >= 2 {
		return cdataMatches[1], true
	}
	return "", false
}
