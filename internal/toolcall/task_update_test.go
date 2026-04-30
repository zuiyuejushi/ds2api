package toolcall

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTaskUpdate(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[1]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[2]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[3]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[4]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[5]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[6]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[7]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[8]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[9]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[completed]]></parameter> 
     <parameter name="taskId"><![CDATA[10]]></parameter> 
   </invoke> 
   <invoke name="TaskUpdate"> 
     <parameter name="status"><![CDATA[in_progress]]></parameter> 
     <parameter name="taskId"><![CDATA[11]]></parameter> 
   </invoke> 
 </tool_calls>`

	calls := ParseToolCalls(xml, []string{"TaskUpdate"})
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

	// 验证调用数量
	if len(calls) != 11 {
		t.Errorf("期望 11 个调用，实际 %d 个", len(calls))
	}

	// 验证每个调用的参数
	// CDATA 包裹的 taskId 是字符串，不是数字
	for i, call := range calls {
		expectedTaskId := fmt.Sprintf("%d", i+1)
		expectedStatus := "completed"
		if i == 10 {
			expectedStatus = "in_progress"
		}

		taskId, ok := call.Input["taskId"].(string)
		if !ok {
			t.Errorf("调用 %d: taskId 不是 string，而是 %T", i+1, call.Input["taskId"])
			continue
		}
		if taskId != expectedTaskId {
			t.Errorf("调用 %d: taskId 期望 %q，实际 %q", i+1, expectedTaskId, taskId)
		}

		status, ok := call.Input["status"].(string)
		if !ok {
			t.Errorf("调用 %d: status 不是 string，而是 %T", i+1, call.Input["status"])
			continue
		}
		if status != expectedStatus {
			t.Errorf("调用 %d: status 期望 %q，实际 %q", i+1, expectedStatus, status)
		}
	}
}
