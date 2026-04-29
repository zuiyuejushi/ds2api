package toolcall

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAskUserQuestion2(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="AskUserQuestion"> 
     <parameter name="questions"> 
       <item> 
         <header><![CDATA[功法规模]]></header> 
         <question><![CDATA[功法系统需要"大改"。第一版修仙赛季中，最小可运行的功法系统应该有多大？]]></question> 
         <multiSelect>false</multiSelect> 
         <options> 
           <item> 
             <label><![CDATA[轻量功法(3-8种)]]></label> 
             <description><![CDATA[3-8种核心功法，每境界1-2种，作为被动加成。复用现有被动技能框架，1周内完成。]]></description> 
           </item> 
           <item> 
             <label><![CDATA[中等功法(15-30种)]]></label> 
             <description><![CDATA[15-30种功法，覆盖5-8个境界，每境界3-5种。有主动/被动区分，部分独立于现有技能系统。2-4周。]]></description> 
           </item> 
           <item> 
             <label><![CDATA[大型功法(50+种)]]></label> 
             <description><![CDATA[50+种功法，完整修炼树，每境界5-10种。独立属性体系（灵力/神识）、独立技能管线、功法升级机制。1月+。]]></description> 
           </item> 
           <item> 
             <label><![CDATA[修炼分支树]]></label> 
             <description><![CDATA[不仅有功法技能，还有修炼路径选择（体修/剑修/丹修），每分支独特的功法树和被动。最大架构改造量。]]></description> 
           </item> 
         </options> 
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
			// 尝试格式化为 JSON 以便查看结构
			jsonBytes, _ := json.MarshalIndent(v, "    ", "  ")
			fmt.Printf("    %s: %s (type: %T)\n", k, jsonBytes, v)
		}
	}
}
