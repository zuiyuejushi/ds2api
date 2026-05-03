package toolcall

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
)

var xmlAttrPattern = regexp.MustCompile(`(?is)\b([a-z0-9_:-]+)\s*=\s*("([^"]*)"|'([^']*)')`)
var xmlToolCallsClosePattern = regexp.MustCompile(`(?is)</tool_calls>`)
var xmlInvokeStartPattern = regexp.MustCompile(`(?is)<invoke\b[^>]*\bname\s*=\s*("([^"]*)"|'([^']*)')`)

func parseXMLToolCalls(text string) []ParsedToolCall {
	wrappers := findXMLElementBlocks(text, "tool_calls")
	if len(wrappers) == 0 {
		repaired := repairMissingXMLToolCallsOpeningWrapper(text)
		if repaired != text {
			wrappers = findXMLElementBlocks(repaired, "tool_calls")
		}
	}
	out := make([]ParsedToolCall, 0, len(wrappers))
	for _, wrapper := range wrappers {
		for _, block := range findXMLElementBlocks(wrapper.Body, "invoke") {
			call, ok := parseSingleXMLToolCall(block)
			if !ok {
				continue
			}
			out = append(out, call)
		}
	}
	// Also parse bare <invoke> tags outside <tool_calls> wrappers
	for _, block := range findXMLElementBlocks(text, "invoke") {
		// Skip invokes already parsed inside wrappers
		if isInsideAnyWrapper(block, wrappers) {
			continue
		}
		call, ok := parseSingleXMLToolCall(block)
		if !ok {
			continue
		}
		out = append(out, call)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isInsideAnyWrapper(block xmlElementBlock, wrappers []xmlElementBlock) bool {
	for _, w := range wrappers {
		if block.Start >= w.Start && block.End <= w.End {
			return true
		}
	}
	return false
}

func repairMissingXMLToolCallsOpeningWrapper(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "<tool_calls") {
		return text
	}

	closeMatches := xmlToolCallsClosePattern.FindAllStringIndex(text, -1)
	if len(closeMatches) == 0 {
		return text
	}
	invokeLoc := xmlInvokeStartPattern.FindStringIndex(text)
	if invokeLoc == nil {
		return text
	}
	closeLoc := closeMatches[len(closeMatches)-1]
	if invokeLoc[0] >= closeLoc[0] {
		return text
	}

	return text[:invokeLoc[0]] + "<tool_calls>" + text[invokeLoc[0]:closeLoc[0]] + "</tool_calls>" + text[closeLoc[1]:]
}

func parseSingleXMLToolCall(block xmlElementBlock) (ParsedToolCall, bool) {
	attrs := parseXMLTagAttributes(block.Attrs)
	name := strings.TrimSpace(html.UnescapeString(attrs["name"]))
	if name == "" {
		return ParsedToolCall{}, false
	}

	inner := strings.TrimSpace(block.Body)
	if strings.HasPrefix(inner, "{") {
		var payload map[string]any
		if err := json.Unmarshal([]byte(inner), &payload); err == nil {
			input := map[string]any{}
			if params, ok := payload["input"].(map[string]any); ok {
				input = params
			}
			if len(input) == 0 {
				if params, ok := payload["parameters"].(map[string]any); ok {
					input = params
				}
			}
			return ParsedToolCall{Name: name, Input: input}, true
		}
	}

	input := map[string]any{}
	for _, paramMatch := range findXMLElementBlocks(inner, "parameter") {
		paramAttrs := parseXMLTagAttributes(paramMatch.Attrs)
		paramName := strings.TrimSpace(html.UnescapeString(decodeUnicodeEscapes(paramAttrs["name"])))
		if paramName == "" {
			continue
		}
		// For parameter values, preserve nested XML structure as string
		// instead of parsing it into a flat map
		value := parseParameterValuePreserveXML(paramMatch.Body)
		appendMarkupValue(input, paramName, value)
	}

	if len(input) == 0 {
		if strings.TrimSpace(inner) != "" {
			return ParsedToolCall{}, false
		}
		return ParsedToolCall{Name: name, Input: map[string]any{}}, true
	}
	return ParsedToolCall{Name: name, Input: input}, true
}

type xmlElementBlock struct {
	Attrs string
	Body  string
	Start int
	End   int
}

func findXMLElementBlocks(text, tag string) []xmlElementBlock {
	if text == "" || tag == "" {
		return nil
	}
	var out []xmlElementBlock
	pos := 0
	for pos < len(text) {
		start, bodyStart, attrs, ok := findXMLStartTagOutsideCDATA(text, tag, pos)
		if !ok {
			break
		}
		closeStart, closeEnd, ok := findMatchingXMLEndTagOutsideCDATA(text, tag, bodyStart)
		if !ok {
			break
		}
		out = append(out, xmlElementBlock{
			Attrs: attrs,
			Body:  text[bodyStart:closeStart],
			Start: start,
			End:   closeEnd,
		})
		pos = closeEnd
	}
	return out
}

func findXMLStartTagOutsideCDATA(text, tag string, from int) (start, bodyStart int, attrs string, ok bool) {
	lower := strings.ToLower(text)
	target := "<" + strings.ToLower(tag)
	for i := maxInt(from, 0); i < len(text); {
		next, advanced, blocked := skipXMLIgnoredSection(lower, i)
		if blocked {
			return -1, -1, "", false
		}
		if advanced {
			i = next
			continue
		}
		if strings.HasPrefix(lower[i:], target) && hasXMLTagBoundary(text, i+len(target)) {
			end := findXMLTagEnd(text, i+len(target))
			if end < 0 {
				return -1, -1, "", false
			}
			return i, end + 1, text[i+len(target) : end], true
		}
		i++
	}
	return -1, -1, "", false
}

func findMatchingXMLEndTagOutsideCDATA(text, tag string, from int) (closeStart, closeEnd int, ok bool) {
	lower := strings.ToLower(text)
	openTarget := "<" + strings.ToLower(tag)
	closeTarget := "</" + strings.ToLower(tag)
	depth := 1
	for i := maxInt(from, 0); i < len(text); {
		next, advanced, blocked := skipXMLIgnoredSection(lower, i)
		if blocked {
			return -1, -1, false
		}
		if advanced {
			i = next
			continue
		}
		if strings.HasPrefix(lower[i:], closeTarget) && hasXMLTagBoundary(text, i+len(closeTarget)) {
			end := findXMLTagEnd(text, i+len(closeTarget))
			if end < 0 {
				return -1, -1, false
			}
			depth--
			if depth == 0 {
				return i, end + 1, true
			}
			i = end + 1
			continue
		}
		if strings.HasPrefix(lower[i:], openTarget) && hasXMLTagBoundary(text, i+len(openTarget)) {
			end := findXMLTagEnd(text, i+len(openTarget))
			if end < 0 {
				return -1, -1, false
			}
			if !isSelfClosingXMLTag(text[:end]) {
				depth++
			}
			i = end + 1
			continue
		}
		i++
	}
	return -1, -1, false
}

func skipXMLIgnoredSection(lower string, i int) (next int, advanced bool, blocked bool) {
	switch {
	case strings.HasPrefix(lower[i:], "<![cdata["):
		end := strings.Index(lower[i+len("<![cdata["):], "]]>")
		if end < 0 {
			return 0, false, true
		}
		return i + len("<![cdata[") + end + len("]]>"), true, false
	case strings.HasPrefix(lower[i:], "<!--"):
		end := strings.Index(lower[i+len("<!--"):], "-->")
		if end < 0 {
			return 0, false, true
		}
		return i + len("<!--") + end + len("-->"), true, false
	default:
		return i, false, false
	}
}

func findXMLTagEnd(text string, from int) int {
	quote := byte(0)
	for i := maxInt(from, 0); i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '>' {
			return i
		}
	}
	return -1
}

func hasXMLTagBoundary(text string, idx int) bool {
	if idx >= len(text) {
		return true
	}
	switch text[idx] {
	case ' ', '\t', '\n', '\r', '>', '/':
		return true
	default:
		return false
	}
}

func isSelfClosingXMLTag(startTag string) bool {
	return strings.HasSuffix(strings.TrimSpace(startTag), "/")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parseXMLTagAttributes(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, m := range xmlAttrPattern.FindAllStringSubmatch(raw, -1) {
		if len(m) < 5 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(m[1]))
		if key == "" {
			continue
		}
		value := m[3]
		if value == "" {
			value = m[4]
		}
		out[key] = value
	}
	return out
}

func parseInvokeParameterValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if value, ok := extractStandaloneCDATA(trimmed); ok {
		// Try to parse CDATA content as JSON literal (bool, number, null)
		// This handles cases like <![CDATA[true]]> or <![CDATA[42]]>
		if parsed, ok := parseJSONLiteralValue(value); ok {
			return parsed
		}
		return decodeUnicodeEscapes(value)
	}
	if parsed := parseStructuredToolCallInput(trimmed); len(parsed) > 0 {
		if len(parsed) == 1 {
			if rawValue, ok := parsed["_raw"].(string); ok {
				// Try to parse raw value as JSON literal for explicit boolean strings
				if parsed, ok := parseJSONLiteralValue(rawValue); ok {
					return parsed
				}
				return decodeUnicodeEscapes(rawValue)
			}
		}
		return parsed
	}
	value := extractRawTagValue(trimmed)
	// Try to parse as JSON literal for explicit boolean strings
	if parsed, ok := parseJSONLiteralValue(value); ok {
		return parsed
	}
	return value
}

// parseParameterValuePreserveXML parses parameter values into structured data (map or array).
// It converts nested XML into JSON-like structure, with repeated tags becoming arrays.
func parseParameterValuePreserveXML(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	// Handle CDATA - extract content and parse if it's a simple literal
	// Parse booleans, null, and pure numbers
	if value, ok := extractStandaloneCDATA(trimmed); ok {
		cleanValue := strings.TrimSpace(value)
		lowerValue := strings.ToLower(cleanValue)

		// Parse boolean/null literals
		switch lowerValue {
		case "true":
			return true
		case "false":
			return false
		case "null":
			return nil
		}

		// Parse pure numbers (optional: check if it's a valid number format)
		if isPureNumber(cleanValue) {
			var num float64
			if err := json.Unmarshal([]byte(cleanValue), &num); err == nil {
				return num
			}
		}

		return value
	}

	// If content looks like XML, parse it into structured data
	if strings.HasPrefix(trimmed, "<") {
		return parseXMLToStructure(trimmed)
	}

	// For non-XML content, use standard processing
	value := extractRawTagValue(trimmed)
	if parsed, ok := parseJSONLiteralValue(value); ok {
		return parsed
	}
	return value
}

// isPureNumber checks if a string represents a pure number (integer or float).
// Allows optional leading minus sign and decimal point.
func isPureNumber(s string) bool {
	if s == "" {
		return false
	}
	// Allow optional leading minus
	start := 0
	if s[0] == '-' {
		if len(s) == 1 {
			return false
		}
		start = 1
	}
	// Check remaining characters are digits or decimal point
	hasDecimal := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if hasDecimal {
				return false // Multiple decimal points
			}
			hasDecimal = true
		} else if c < '0' || c > '9' {
			return false // Not a digit
		}
	}
	return true
}

// parseXMLToStructure converts XML content into a structured map or array.
// Uses Go's standard xml.Decoder for proper nested XML parsing.
// Delegates to parseXMLFragmentValue from toolcalls_xml.go.
func parseXMLToStructure(xmlContent string) any {
	trimmed := strings.TrimSpace(xmlContent)
	if trimmed == "" {
		return ""
	}

	// Use parseXMLFragmentValue from toolcalls_xml.go for proper XML parsing
	if value, ok := parseXMLFragmentValue(trimmed); ok {
		return value
	}

	return trimmed
}
