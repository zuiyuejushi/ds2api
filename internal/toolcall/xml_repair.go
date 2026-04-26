package toolcall

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// RepairXMLToolCalls repairs malformed XML tool calls and returns standardized format
// Three-step process:
// 1. Context Pre-processing: Fix structural issues (missing wrappers, misplaced attributes)
// 2. Tolerant Parsing: Use lenient XML parser to handle format errors
// 3. Structure Standardization: Convert to canonical format
func RepairXMLToolCalls(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}

	// Step 1: Context Pre-processing
	preprocessed := preprocessContext(trimmed)

	// Step 2: Tolerant Parsing & Step 3: Structure Standardization
	standardized := standardizeStructure(preprocessed)
	if standardized == "" {
		return "", false
	}

	return standardized, true
}

// Step 1: Context Pre-processing
// - Fix missing <tool_calls> wrapper
// - Fix misplaced attributes on <parameter> tags
// - Convert legacy <tools> format to standard format
// - Fix unclosed CDATA sections
func preprocessContext(text string) string {
	result := text

	// Fix 1: Fix unclosed CDATA sections
	// Pattern: <![CDATA[...]</parameter> (missing ]]>)
	result = fixUnclosedCDATA(result)

	// Fix 2: Convert legacy <tools> format to <tool_calls>
	result = convertLegacyToolsFormat(result)

	// Fix 3: Wrap bare <invoke> with <tool_calls>
	// Pattern: <invoke ...> ... </invoke> without surrounding <tool_calls>
	result = wrapBareInvokes(result)

	// Fix 4: Convert attribute-style parameters to nested structure
	// Pattern: <parameter name="file_path" x="..." limit="N">
	// Convert to: <parameter name="file_path">...</parameter> with nested values
	result = convertAttributeParams(result)

	return result
}

// fixUnclosedCDATA fixes CDATA sections that are not properly closed
// Input:  <![CDATA[value]</parameter>
// Output: <![CDATA[value]]></parameter>
func fixUnclosedCDATA(text string) string {
	// Pattern: <![CDATA[...]</parameter> or <![CDATA[...]</invoke> etc.
	// Find CDATA start that doesn't have proper ]]> close before next tag
	cdataStartPattern := regexp.MustCompile(`(?i)<!\[CDATA\[`)

	result := text
	offset := 0

	for {
		loc := cdataStartPattern.FindStringIndex(result[offset:])
		if loc == nil {
			break
		}
		contentStart := offset + loc[1]

		// Find the next tag after CDATA start
		nextTagPattern := regexp.MustCompile(`(?i)<\s*/\s*(?:parameter|invoke|tool_calls)\s*>`)
		nextTagLoc := nextTagPattern.FindStringIndex(result[contentStart:])
		if nextTagLoc == nil {
			break
		}
		nextTagIdx := contentStart + nextTagLoc[0]

		// Check if there's a proper ]]> close between contentStart and nextTagIdx
		content := result[contentStart:nextTagIdx]
		
		// Find ]]> close
		closeIdx := strings.Index(content, "]]>")
		if closeIdx < 0 {
			// CDATA is not closed, add ]]> before the tag
			result = result[:nextTagIdx] + "]]>" + result[nextTagIdx:]
			offset = nextTagIdx + 3 // Skip past the ]]> we just added
		} else {
			offset = contentStart + closeIdx + 3
		}
	}

	return result
}

// convertLegacyToolsFormat converts <tools><tool_call>...</tool_call></tools> to <tool_calls><invoke>...</invoke></tool_calls>
func convertLegacyToolsFormat(text string) string {
	// Check if it's legacy format
	if !strings.Contains(strings.ToLower(text), "<tools>") {
		return text
	}

	result := text

	// Replace <tool_name>...</tool_name> with name attribute on <tool_call> first
	// This needs to happen before we rename tool_call to invoke
	toolNamePattern := regexp.MustCompile(`(?is)<tool_call\s*>(.*?)<tool_name\s*>(.*?)</tool_name\s*>(.*?)</tool_call\s*>`)
	result = toolNamePattern.ReplaceAllStringFunc(result, func(match string) string {
		submatches := toolNamePattern.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		before := submatches[1]
		toolName := strings.TrimSpace(submatches[2])
		after := submatches[3]
		return fmt.Sprintf(`<tool_call name="%s">%s%s</tool_call>`, toolName, before, after)
	})

	// Replace <param>...</param> with <parameter name="...">...</parameter>
	paramPattern := regexp.MustCompile(`(?is)<param\s*>(.*?)</param\s*>`)
	result = paramPattern.ReplaceAllStringFunc(result, func(match string) string {
		submatches := paramPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		paramContent := strings.TrimSpace(submatches[1])
		// Try to parse as JSON to extract parameter name
		var jsonParams map[string]any
		if err := json.Unmarshal([]byte(paramContent), &jsonParams); err == nil && len(jsonParams) > 0 {
			// Build multiple parameter tags
			var params []string
			for key, value := range jsonParams {
				params = append(params, fmt.Sprintf(`<parameter name="%s">%v</parameter>`, key, value))
			}
			return strings.Join(params, "")
		}
		// If not JSON, treat as raw value with generic name
		return fmt.Sprintf(`<parameter name="value">%s</parameter>`, paramContent)
	})

	// Replace <tools> with <tool_calls>
	result = regexp.MustCompile(`(?is)<tools\s*>`).ReplaceAllString(result, "<tool_calls>")
	result = regexp.MustCompile(`(?is)</tools\s*>`).ReplaceAllString(result, "</tool_calls>")

	// Replace <tool_call name="..."> with <invoke name="...">
	result = regexp.MustCompile(`(?is)<tool_call(?:\s+name="([^"]*)")?\s*>`).ReplaceAllString(result, `<invoke name="$1">`)
	result = regexp.MustCompile(`(?is)</tool_call\s*>`).ReplaceAllString(result, "</invoke>")

	return result
}

// wrapBareInvokes wraps bare <invoke> tags with <tool_calls>
func wrapBareInvokes(text string) string {
	// Check if already has <tool_calls>
	if strings.Contains(strings.ToLower(text), "<tool_calls") {
		return text
	}

	// Check if has bare <invoke>
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "<invoke") {
		return text
	}

	// Wrap entire content with <tool_calls>
	return "<tool_calls>" + text + "</tool_calls>"
}

// convertAttributeParams converts attribute-style parameters to nested structure
// Input:  <parameter name="file_path" x="c:\path" limit="50">
// Output: <parameter name="file_path">c:\path</parameter><limit>50</limit>
func convertAttributeParams(text string) string {
	// Pattern to match <parameter name="..." with additional attributes
	// Capture: name attribute and any other attributes
	paramPattern := regexp.MustCompile(`(?i)<parameter\s+name\s*=\s*["']([^"']+)["']\s*([^>]*)>`)

	result := paramPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := paramPattern.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		paramName := submatches[1]
		extraAttrs := submatches[2]

		// Extract value from x="...", value="...", or path="..." attribute
		valuePattern := regexp.MustCompile(`(?i)(?:x|value|path|file_path)\s*=\s*["']([^"']+)["']`)
		valueMatch := valuePattern.FindStringSubmatch(extraAttrs)

		var paramValue string
		if len(valueMatch) >= 2 {
			paramValue = valueMatch[1]
		}

		// Build replacement: <parameter name="...">value</parameter>
		if paramValue != "" {
			return fmt.Sprintf(`<parameter name="%s">%s</parameter>`, paramName, paramValue)
		}

		// If no value found, return original but ensure it's properly formed
		return fmt.Sprintf(`<parameter name="%s">`, paramName)
	})

	// Also extract numeric attributes like limit="50" offset="38" as separate parameters
	numericPattern := regexp.MustCompile(`(?i)(limit|offset|timeout)\s*=\s*["'](\d+)["']`)

	// Find all numeric attributes and append them as separate parameter tags
	numericMatches := numericPattern.FindAllStringSubmatch(result, -1)
	for _, match := range numericMatches {
		if len(match) >= 3 {
			attrName := match[1]
			attrValue := match[2]
			// Append as separate parameter tag before closing </invoke>
			paramTag := fmt.Sprintf(`<parameter name="%s">%s</parameter>`, attrName, attrValue)
			// Insert before </invoke>
			result = regexp.MustCompile(`(?i)(</invoke>)`).ReplaceAllString(result, paramTag+`$1`)
		}
	}

	return result
}

// Step 2 & 3: Tolerant Parsing + Structure Standardization
// Uses Go's encoding/xml with strict=false mode
func standardizeStructure(text string) string {
	// Try to parse with tolerant settings
	type Parameter struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	}

	type Invoke struct {
		Name       string      `xml:"name,attr"`
		Parameters []Parameter `xml:"parameter"`
	}

	type ToolCalls struct {
		Invokes []Invoke `xml:"invoke"`
	}

	var toolCalls ToolCalls

	// Create a decoder with strict mode disabled
	decoder := xml.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	if err := decoder.Decode(&toolCalls); err != nil {
		// If parsing fails, return empty
		return ""
	}

	// Reconstruct standardized XML
	var buf bytes.Buffer
	buf.WriteString("<tool_calls>")

	for _, invoke := range toolCalls.Invokes {
		if invoke.Name == "" {
			continue
		}

		buf.WriteString(fmt.Sprintf(`<invoke name="%s">`, escapeXML(invoke.Name)))

		for _, param := range invoke.Parameters {
			if param.Name == "" {
				continue
			}
			buf.WriteString(fmt.Sprintf(`<parameter name="%s">%s</parameter>`,
				escapeXML(param.Name),
				escapeXML(param.Value)))
		}

		buf.WriteString("</invoke>")
	}

	buf.WriteString("</tool_calls>")

	result := buf.String()
	if result == "<tool_calls></tool_calls>" {
		return ""
	}

	return result
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// TryRepairAndParse attempts to repair malformed XML and parse it
func TryRepairAndParse(text string, toolNames []string) ([]ParsedToolCall, bool) {
	repaired, ok := RepairXMLToolCalls(text)
	if !ok {
		return nil, false
	}

	parsed := parseXMLToolCalls(repaired)
	if len(parsed) == 0 {
		return nil, false
	}

	// Filter by tool names
	var filtered []ParsedToolCall
	for _, call := range parsed {
		if call.Name != "" {
			filtered = append(filtered, call)
		}
	}

	return filtered, len(filtered) > 0
}
