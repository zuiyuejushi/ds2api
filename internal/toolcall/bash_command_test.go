package toolcall

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBashCommand(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="Bash"> 
     <parameter name="command"><![CDATA[which certbot acme.sh openssl 2>/dev/null; dpkg -l certbot 2>/dev/null | grep -E '^ii'; ls ~/.acme.sh/acme.sh 2>/dev/null; echo "---"; cat /etc/os-release | head -4]]></parameter> 
     <parameter name="description"><![CDATA[Check installed ACME tools and OS]]></parameter> 
   </invoke> 
   <invoke name="Bash"> 
     <parameter name="command"><![CDATA[ls /etc/letsencrypt/live/ 2>/dev/null; ls /etc/nginx/sites-enabled/ 2>/dev/null; ls /etc/apache2/sites-enabled/ 2>/dev/null; which nginx apache2 caddy 2>/dev/null]]></parameter> 
     <parameter name="description"><![CDATA[Check existing certs and web servers]]></parameter> 
   </invoke> 
 </tool_calls>`

	calls := ParseToolCalls(xml, []string{"Bash"})
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
	if len(calls) != 2 {
		t.Errorf("期望 2 个调用，实际 %d 个", len(calls))
	}
}
