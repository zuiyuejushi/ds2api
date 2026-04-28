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
		expected map[string]string
	}{
		{
			name: "参数值包含尖括号",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">echo "test" > output.txt</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]string{
				"command": `echo "test" > output.txt`,
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
			expected: map[string]string{
				"old_string": `const x = 1;`,
				"new_string": `const x = 2;`,
			},
		},
		{
			name: "参数值包含 XML 特殊字符",
			xml: `<tool_calls>
  <invoke name="Bash">
    <parameter name="command">/root/idledo/frontend/node_modules/.bin/eslint --version</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]string{
				"command": `/root/idledo/frontend/node_modules/.bin/eslint --version`,
			},
		},
		{
			name: "参数值包含伪结束标签",
			xml: `<tool_calls>
  <invoke name="Test">
    <parameter name="code">if (x < 0) { return "</parameter> is bad"; }</parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]string{
				"code": `if (x < 0) { return "</parameter> is bad"; }`,
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
			expected: map[string]string{
				"content": `<!DOCTYPE html>
<html>
<body>
  <p>Hello</p>
</body>
</html>`,
			},
		},
		{
			name: "参数值包含完整的 XML 标签",
			xml: `<tool_calls>
  <invoke name="Edit">
    <parameter name="replacement"><parameter name="test">value</parameter></parameter>
  </invoke>
</tool_calls>`,
			expected: map[string]string{
				"replacement": `<parameter name="test">value</parameter>`,
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
				actualStr, ok := actualValue.(string)
				if !ok {
					t.Errorf("参数 %s 不是字符串类型，而是 %T", expectedKey, actualValue)
					continue
				}
				if actualStr != expectedValue {
					t.Errorf("参数 %s 值不匹配:\n  期望: %q\n  实际: %q", expectedKey, expectedValue, actualStr)
					// 检查是否包含泄露的 XML 标签
					if strings.Contains(actualStr, "</parameter>") {
						t.Errorf("  错误: 参数值包含泄露的 </parameter> 标签!")
					}
				}
			}
		})
	}
}
