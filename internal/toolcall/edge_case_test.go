package toolcall

import (
	"fmt"
	"testing"
)

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		xml  string
	}{
		{
			name: "Nested CDATA-like content",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="code"><![CDATA[function test() { return "<![CDATA[inner]]>"; }]]></parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Unclosed CDATA",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="broken"><![CDATA[never closes</parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Special chars in parameter",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="cmd">echo "hello" && cat < file.txt | grep "test"</parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Boolean-like strings",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="val1">true</parameter>
    <parameter name="val2">True</parameter>
    <parameter name="val3">TRUE</parameter>
    <parameter name="val4">false</parameter>
    <parameter name="val5">yes</parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Empty parameters",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="empty"></parameter>
    <parameter name="empty_cdata"><![CDATA[]]></parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "XML in CDATA",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="xml_content"><![CDATA[<nested><tag>value</tag></nested>]]></parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Newlines in CDATA",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="multiline"><![CDATA[line1
line2
line3]]></parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Mixed case tags",
			xml: `<TOOL_CALLS>
  <INVOKE name="test">
    <PARAMETER name="val">test</PARAMETER>
  </INVOKE>
</TOOL_CALLS>`,
		},
		{
			name: "Parameter with only whitespace",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="spaces">   </parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "Unicode in parameter names",
			xml: `<tool_calls>
  <invoke name="test">
    <parameter name="\u4e2d\u6587">value</parameter>
  </invoke>
</tool_calls>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := ParseToolCalls(tt.xml, []string{"test"})
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Input XML length: %d\n", len(tt.xml))
			fmt.Printf("Parsed calls: %d\n", len(calls))
			if len(calls) > 0 {
				for i, call := range calls {
					fmt.Printf("Call %d - Name: %s, Params: %v\n", i+1, call.Name, call.Input)
				}
			}
		})
	}
}
