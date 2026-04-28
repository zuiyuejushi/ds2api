package toolcall

import (
	"fmt"
	"testing"
)

func TestVerifyXMLParsing(t *testing.T) {
	xml := `<tool_calls>
  <invoke name="Agent">
    <parameter name="description"><![CDATA[Create PoolInterface and adapt pool]]></parameter>
    <parameter name="prompt"><![CDATA[You are implementing Step 1 and Step 2...]]></parameter>
    <parameter name="run_in_background"><![CDATA[true]]></parameter>
    <parameter name="subagent_type"><![CDATA[oh-my-claudecode:executor]]></parameter>
  </invoke>
</tool_calls>`

	calls := ParseToolCalls(xml, []string{"Agent"})
	
	if len(calls) == 0 {
		t.Fatal("No tool calls parsed")
	}
	
	call := calls[0]
	fmt.Printf("Tool Name: %s\n", call.Name)
	fmt.Printf("Input Parameters:\n")
	
	for key, value := range call.Input {
		fmt.Printf("  %s: %v (type: %T)\n", key, value, value)
	}
	
	// Verify run_in_background is parsed as boolean
	if runInBg, ok := call.Input["run_in_background"]; ok {
		switch v := runInBg.(type) {
		case bool:
			fmt.Printf("\n✓ run_in_background is correctly parsed as bool: %v\n", v)
		case string:
			fmt.Printf("\n✗ run_in_background is still a string: %q\n", v)
		default:
			fmt.Printf("\n✗ run_in_background has unexpected type: %T\n", v)
		}
	} else {
		fmt.Printf("\n✗ run_in_background not found in input\n")
	}
}
