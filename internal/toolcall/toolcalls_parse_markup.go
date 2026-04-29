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
	if len(wrappers) == 0 {
		return nil
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
	if len(out) == 0 {
		return nil
	}
	return out
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

	// Handle CDATA - extract and process content
	if value, ok := extractStandaloneCDATA(trimmed); ok {
		if parsed, ok := parseJSONLiteralValue(value); ok {
			return parsed
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

// parseXMLToStructure converts XML content into a structured map or array.
// Repeated tags with the same name are converted to arrays.
func parseXMLToStructure(xmlContent string) any {
	trimmed := strings.TrimSpace(xmlContent)
	if trimmed == "" {
		return ""
	}

	// Check if this is a single tag or multiple tags at root level
	topLevelTags := findTopLevelTags(trimmed)
	if len(topLevelTags) == 0 {
		return trimmed
	}

	// If single tag, parse it as an object
	if len(topLevelTags) == 1 {
		return parseXMLNode(trimmed)
	}

	// Multiple top-level tags with same name -> array
	firstTagName := getTagName(topLevelTags[0])
	allSameName := true
	for _, tag := range topLevelTags {
		if getTagName(tag) != firstTagName {
			allSameName = false
			break
		}
	}

	if allSameName {
		result := make([]any, 0, len(topLevelTags))
		for _, tag := range topLevelTags {
			result = append(result, parseXMLNode(tag))
		}
		return result
	}

	// Mixed tags -> map
	result := make(map[string]any)
	for _, tag := range topLevelTags {
		node := parseXMLNode(tag)
		if nodeMap, ok := node.(map[string]any); ok {
			for k, v := range nodeMap {
				appendXMLValue(result, k, v)
			}
		}
	}
	return result
}

// findTopLevelTags extracts top-level XML tags from content
func findTopLevelTags(content string) []string {
	var tags []string
	pos := 0
	contentLen := len(content)

	for pos < contentLen {
		// Skip whitespace
		for pos < contentLen && (content[pos] == ' ' || content[pos] == '\t' || content[pos] == '\n' || content[pos] == '\r') {
			pos++
		}
		if pos >= contentLen {
			break
		}

		// Check if this is a tag start
		if content[pos] != '<' || pos+1 >= contentLen || content[pos+1] == '/' || content[pos+1] == '!' {
			// Not a start tag, skip to next
			pos++
			continue
		}

		// Find tag name
		tagStart := pos
		tagEnd := pos + 1
		for tagEnd < contentLen && content[tagEnd] != ' ' && content[tagEnd] != '>' && content[tagEnd] != '/' {
			tagEnd++
		}
		tagName := content[tagStart+1 : tagEnd]

		// Find matching end tag
		endTag := "</" + tagName + ">"
		endPos := strings.Index(content[tagEnd:], endTag)
		if endPos < 0 {
			pos++
			continue
		}
		endPos += tagEnd + len(endTag)

		// Extract the complete tag
		tags = append(tags, content[tagStart:endPos])
		pos = endPos
	}

	return tags
}

// getTagName extracts tag name from a complete tag string
func getTagName(tag string) string {
	tag = strings.TrimSpace(tag)
	if len(tag) < 2 || tag[0] != '<' {
		return ""
	}
	end := 1
	for end < len(tag) && tag[end] != ' ' && tag[end] != '>' && tag[end] != '/' {
		end++
	}
	return strings.ToLower(tag[1:end])
}

// parseXMLNode parses a single XML node into a map
func parseXMLNode(node string) any {
	node = strings.TrimSpace(node)
	if node == "" {
		return ""
	}

	// Extract tag name and body
	if len(node) < 2 || node[0] != '<' {
		return node
	}

	// Find tag end
	tagEnd := 1
	for tagEnd < len(node) && node[tagEnd] != ' ' && node[tagEnd] != '>' {
		tagEnd++
	}
	tagName := strings.ToLower(node[1:tagEnd])

	// Find body start
	bodyStart := tagEnd
	for bodyStart < len(node) && node[bodyStart] != '>' {
		bodyStart++
	}
	if bodyStart >= len(node) {
		return node
	}
	bodyStart++ // Skip '>'

	// Find end tag
	endTag := "</" + tagName + ">"
	endPos := strings.LastIndex(strings.ToLower(node), endTag)
	if endPos < 0 {
		return node
	}

	body := node[bodyStart:endPos]
	body = strings.TrimSpace(body)

	// If body contains nested tags, parse them into a map
	if strings.Contains(body, "<") {
		parsedBody := parseXMLToStructure(body)
		// Return as a map with tag name as key
		return map[string]any{tagName: parsedBody}
	}

	// Simple text content - return as map with tag name as key
	return map[string]any{tagName: body}
}

// appendXMLValue appends a value to a map, converting to array if key already exists
func appendXMLValue(m map[string]any, key string, value any) {
	if existing, ok := m[key]; ok {
		// Convert to array
		switch v := existing.(type) {
		case []any:
			m[key] = append(v, value)
		default:
			m[key] = []any{v, value}
		}
	} else {
		m[key] = value
	}
}


