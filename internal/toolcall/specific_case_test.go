package toolcall

import (
	"fmt"
	"strings"
	"testing"
)

func TestSpecificErrorCase(t *testing.T) {
	// 这是用户提供的具体报错例子
	xml := `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">/root/idledo/frontend/node_modules/.bin/eslint --version</parameter>
    <parameter name="description"><![CDATA[Verify frontend ESLint installation]]></parameter>
  </invoke>
</tool_calls>`

	calls := ParseToolCalls(xml, []string{"Bash"})
	
	fmt.Printf("解析结果:\n")
	fmt.Printf("调用数量: %d\n\n", len(calls))
	
	if len(calls) > 0 {
		call := calls[0]
		fmt.Printf("工具名: %s\n", call.Name)
		fmt.Printf("参数:\n")
		for k, v := range call.Input {
			strVal, ok := v.(string)
			if !ok {
				fmt.Printf("  %s: %v (type: %T)\n", k, v, v)
			} else {
				fmt.Printf("  %s: %q\n", k, strVal)
				// 检查是否包含泄露的 XML 标签
				if strings.Contains(strVal, "</parameter>") {
					t.Errorf("错误: 参数 %s 包含泄露的 </parameter> 标签!", k)
				}
				if strings.Contains(strVal, "<parameter") {
					t.Errorf("错误: 参数 %s 包含泄露的 <parameter 标签!", k)
				}
			}
		}
	}
	
	// 验证期望值
	if len(calls) != 1 {
		t.Fatalf("期望 1 个调用，实际 %d 个", len(calls))
	}
	
	call := calls[0]
	if call.Name != "Bash" {
		t.Errorf("期望工具名 Bash，实际 %s", call.Name)
	}
	
	// 验证 command 参数
	cmdVal, ok := call.Input["command"].(string)
	if !ok {
		t.Errorf("command 不是字符串类型: %T", call.Input["command"])
	} else {
		expected := "/root/idledo/frontend/node_modules/.bin/eslint --version"
		if cmdVal != expected {
			t.Errorf("command 值不匹配:\n期望: %q\n实际: %q", expected, cmdVal)
		}
	}
	
	// 验证 description 参数
	descVal, ok := call.Input["description"].(string)
	if !ok {
		t.Errorf("description 不是字符串类型: %T", call.Input["description"])
	} else {
		expected := "Verify frontend ESLint installation"
		if descVal != expected {
			t.Errorf("description 值不匹配:\n期望: %q\n实际: %q", expected, descVal)
		}
	}
}

func TestMalformedXMLCase(t *testing.T) {
	// 测试可能的真实错误情况 - 标签未正确闭合
	testCases := []struct {
		name string
		xml  string
	}{
		{
			name: "参数值中包含未转义的 <",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">echo "test" && if [ 1 < 2 ]; then echo ok; fi</parameter>
  </invoke>
</tool_calls>`,
		},
		{
			name: "缺少结束标签",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">echo test
  </invoke>
</tool_calls>`,
		},
		{
			name: "嵌套的 parameter 标签",
			xml: `<tool_calls>
  <invoke name="Edit">
    <parameter name="old_string"><parameter name="nested">value</parameter></parameter>
  </invoke>
</tool_calls>`,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			calls := ParseToolCalls(tc.xml, []string{"Bash", "Edit"})
			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("解析到 %d 个调用\n", len(calls))
			if len(calls) > 0 {
				for i, call := range calls {
					fmt.Printf("调用 %d: %s\n", i+1, call.Name)
					for k, v := range call.Input {
						fmt.Printf("  %s: %v\n", k, v)
					}
				}
			}
		})
	}
}
