package toolcall

import (
	"fmt"
	"testing"
)

func TestNestedXMLStructure(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="AskUserQuestion"> 
     <parameter name="questions"> 
       <item> 
         <header>功法范围</header> 
         <question>功法系统需要"大改"。第一版修仙赛季中，最小可运行的功法系统应该有多大规模？</question> 
         <multiSelect>false</multiSelect> 
         <options> 
           <item> 
             <label>小规模</label> 
             <description>3-5种功法，每境界1-2种，作为被动加成（如提升灵力、增加暴击率）。复用现有被动技能框架，1周内完成。</description> 
           </item> 
           <item> 
             <label>中等规模</label> 
             <description>15-20种功法，覆盖5大境界，每境界3-4种可选。有主动/被动区分，独立于现有技能系统但共享战斗管线。</description> 
           </item> 
           <item> 
             <label>大规模</label> 
             <description>50+种功法，完整的修炼树，每境界5-10种可选。独立属性体系（灵力/神识）、独立技能管线、功法升级机制。</description> 
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
			fmt.Printf("    %s: %v (type: %T)\n", k, v, v)
		}
	}
}
