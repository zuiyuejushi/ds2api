package toolcall

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTaskOutput(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="TaskOutput"> 
     <parameter name="block"><![CDATA[false]]></parameter> 
     <parameter name="task_id"><![CDATA[a53c5203a4597f056]]></parameter> 
     <parameter name="timeout">5000</parameter> 
   </invoke> 
   <invoke name="TaskOutput"> 
     <parameter name="block"><![CDATA[false]]></parameter> 
     <parameter name="task_id"><![CDATA[a4ec8d17c4edab8e4]]></parameter> 
     <parameter name="timeout">5000</parameter> 
   </invoke> 
   <invoke name="TaskOutput"> 
     <parameter name="block"><![CDATA[false]]></parameter> 
     <parameter name="task_id"><![CDATA[aa49a14312437cf96]]></parameter> 
     <parameter name="timeout">5000</parameter> 
   </invoke> 
 </tool_calls>`

	calls := ParseToolCalls(xml, []string{"TaskOutput"})
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
