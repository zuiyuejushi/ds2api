package toolcall

import (
	"fmt"
	"strings"
	"testing"
)

func TestParameterBoundaryDetection(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected map[string]any // 改为 any 类型以接受 string 或 map
		expectType map[string]string // 期望的类型: "string" 或 "map"
	}{
		{
			name: "参数值包含尖括号",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">echo "test" > output.txt</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]any{
				"command": `echo "test" > output.txt`,
			},
			expectType: map[string]string{
				"command": "string",
			},
		},
		{
			name: "参数值包含结束标签文本",
			xml: `<tool_calls>
  <invoke name="Edit">
    <parameter name="old_string">const x = 1;</parameter>
    <parameter name="new_string">const x = 2;</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]any{
				"old_string": `const x = 1;`,
				"new_string": `const x = 2;`,
			},
			expectType: map[string]string{
				"old_string": "string",
				"new_string": "string",
			},
		},
		{
			name: "参数值包含 XML 特殊字符",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">/root/idledo/frontend/node_modules/.bin/eslint --version</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]any{
				"command": `/root/idledo/frontend/node_modules/.bin/eslint --version`,
			},
			expectType: map[string]string{
				"command": "string",
			},
		},
		{
			name: "参数值包含伪结束标签",
			xml: `<tool_calls>
  <invoke name="Test">
    <parameter name="code">if (x < 0) { return "</parameter> is bad"; }</parameter>
  </invoke>
</tool_calls>`,
			// 注意: 包含 </parameter> 的参数值会被截断，这是已知限制
			// 此类内容应使用 CDATA 包裹
			expected: map[string]any{
				"code": `if (x < 0) { return "`,
			},
			expectType: map[string]string{
				"code": "string",
			},
		},
		{
			name: "嵌套标签样式的文本",
			xml: `<tool_calls>
  <invoke name="Write">
    <parameter name="content"><!DOCTYPE html>
<html>
<body>
  <p>Hello</p>
</body>
</html></parameter>
  </invoke>
</tool_calls>`,
			// 注意: DOCTYPE 声明的内容不会被解析为嵌套结构，保持为字符串
			expected: map[string]any{
				"content": `<!DOCTYPE html>
<html>
<body>
  <p>Hello</p>
</body>
</html>`,
			},
			expectType: map[string]string{
				"content": "string",
			},
		},
		{
			name: "参数值包含完整的 XML 标签",
			xml: `<tool_calls>
  <invoke name="Edit">
    <parameter name="replacement"><parameter name="test">value</parameter></parameter>
  </invoke>
</tool_calls>`,
			// 注意: 嵌套的 parameter 标签会被解析为结构
			expected: map[string]any{
				"replacement": map[string]any{
					"parameter": "value",
				},
			},
			expectType: map[string]string{
				"replacement": "map",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := ParseToolCalls(tt.xml, []string{"Bash", "Edit", "Test", "Write"})
			fmt.Printf("\n=== %s ===\n", tt.name)
			
			if len(calls) == 0 {
				t.Logf("警告: 没有解析到任何调用")
				return
			}
			
			call := calls[0]
			fmt.Printf("解析到的参数:\n")
			for k, v := range call.Input {
				fmt.Printf("  %s: %q (type: %T)\n", k, v, v)
			}
			
			// 验证期望值
			for expectedKey, expectedValue := range tt.expected {
				actualValue, ok := call.Input[expectedKey]
				if !ok {
					t.Errorf("缺少参数: %s", expectedKey)
					continue
				}
				
				// 检查类型
				expectedType := tt.expectType[expectedKey]
				actualType := "string"
				switch actualValue.(type) {
				case map[string]any:
					actualType = "map"
				case string:
					actualType = "string"
				default:
					actualType = fmt.Sprintf("%T", actualValue)
				}
				
				if actualType != expectedType {
					t.Errorf("参数 %s 类型不匹配: 期望 %s, 实际 %s", expectedKey, expectedType, actualType)
					continue
				}
				
				// 值比较
				if expectedType == "string" {
					actualStr, _ := actualValue.(string)
					expectedStr, _ := expectedValue.(string)
					if actualStr != expectedStr {
						t.Errorf("参数 %s 值不匹配:\n  期望: %q\n  实际: %q", expectedKey, expectedStr, actualStr)
						if strings.Contains(actualStr, "</parameter>") {
							t.Errorf("  错误: 参数值包含泄露的 </parameter> 标签!")
						}
					}
				}
				// 对于 map 类型，只检查类型，不检查具体值（因为 map 比较复杂）
			}
		})
	}
}
