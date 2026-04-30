package toolcall

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestMcpState(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="mcp__plugin_oh-my-claudecode_t__state_list_active"> 
     <parameter name="workingDirectory"><![CDATA[/root/idledo]]></parameter> 
     <parameter name="session_id"><![CDATA[abbcfc96-037f-44e0-beaa-e8ef4f6017b2]]></parameter> 
   </invoke> 
   <invoke name="Bash"> 
     <parameter name="command"><![CDATA[npx tsc --noEmit 2>&1; echo "BUILD_EXIT=$?"]]></parameter> 
     <parameter name="description"><![CDATA[Final build verification]]></parameter> 
     <parameter name="timeout"><![CDATA[120000]]></parameter> 
   </invoke> 
 </tool_calls>`

	calls := ParseToolCalls(xml, []string{"mcp__plugin_oh-my-claudecode_t__state_list_active", "Bash"})
	fmt.Printf("解析结果:\n")
	fmt.Printf("调用数量: %d\n\n", len(calls))

	if len(calls) == 0 {
		t.Fatal("没有解析到任何调用")
	}

	for i, call := range calls {
		fmt.Printf("调用 %d:\n", i+1)
		fmt.Printf("  工具名: %s\n", call.Name)
		fmt.Printf("  参数:\n")
		for k, v := range call.Input {
			jsonBytes, _ := json.MarshalIndent(v, "    ", "  ")
			fmt.Printf("    %s: %s (type: %T)\n", k, jsonBytes, v)
		}
	}
}
