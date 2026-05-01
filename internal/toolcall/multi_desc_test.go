package toolcall

import (
	"fmt"
	"testing"
)

func TestMultiDescription(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="AskUserQuestion"> 
     <parameter name="questions"> 
       <item> 
         <description><![CDATA[3-5种功法，每境界1-2种，作为被动加成。复用现有被动技能框架，约1周完成。]]></description> 
         <description><![CDATA[15-20种功法，覆盖5大境界，有主动/被动区分。共享战斗管线但独立数据表。]]></description> 
         <description><![CDATA[50+种功法，完整修炼树，每境界5-10种。独立属性体系（灵力/神识），全新技能管线。]]></description> 
         <description><![CDATA[我逐一给出具体数字（请在下方自由输入）]]></description> 
         <multiSelect>false</multiSelect> 
         <question><![CDATA[功法系统需要"大改"。第一版修仙赛季中，最小可运行的功法系统应该有多大规模？]]></question> 
       </item> 
     </parameter> 
   </invoke> 
 </tool_calls>`

	calls := ParseToolCalls(xml, []string{"AskUserQuestion"})
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
			fmt.Printf("    %s: %v (type: %T)\n", k, v, v)
		}
	}
}
