package toolcall

import (
	"fmt"
	"testing"
)

func TestAskUserQuestion(t *testing.T) {
	xml := `<tool_calls> 
   <invoke name="AskUserQuestion"> 
     <parameter name="questions"> 
       <item> 
         <question>第一版修仙赛季中，功法系统的最小规模应该是多少种功法？</question> 
         <header>功法规模</header> 
         <multiSelect>false</multiSelect> 
         <options> 
           <item> 
             <label>3-5种被动加成</label> 
             <description>每境界1-2种，纯被动属性加成（如提升灵力/暴击率）。复用现有被动技能框架，1周内完成。</description> 
           </item> 
           <item> 
             <label>15-20种完整功法</label> 
             <description>覆盖5大境界，每境界3-4种可选。有主动+被动区分，扩建现有技能框架，2-3周。</description> 
           </item> 
           <item> 
             <label>50+种功法修炼树</label> 
             <description>完整修炼树，每境界5-10种可选。独立属性体系、独立技能管线、功法升级机制。1个月+。</description> 
           </item> 
           <item> 
             <label>我先逐一给出具体数字</label> 
             <description>为每个系统（妖兽/法器/境界/炼制/功法）分别给出最小可行参数</description> 
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
