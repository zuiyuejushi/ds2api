package main

import (
	"fmt"
	"ds2api/internal/toolcall"
)

func main() {
	text := `<tool_calls> 
   <invoke name="TaskCreate"> 
     <parameter name="description"><![CDATA[Delete dead code: helpers_test.ts and rateLimitStore.ts.bak]]></parameter> 
     <parameter name="prompt"><![CDATA[Delete backend/src/utils/helpers_test.ts and backend/src/utils/rateLimitStore.ts.bak - dead code with no imports anywhere]]></parameter> 
     <parameter name="subagent_type"><![CDATA[general-purpose]]></parameter> 
   </invoke> 
   <invoke name="TaskCreate"> 
     <parameter name="description"><![CDATA[Create frontend/src/utils/gameDataLabels.ts with getClassName/getRaceName/getMapName]]></parameter> 
     <parameter name="prompt"><![CDATA[Create shared utility with getClassName(), getRaceName(), getMapName() extracted from 4 page files. Then update CharacterList.tsx, CharacterCreate.tsx, Ranking.tsx, Battle.tsx to import from new shared file instead of local definitions.]]></parameter> 
     <parameter name="subagent_type"><![CDATA[general-purpose]]></parameter> 
   </invoke> 
   <invoke name="TaskCreate"> 
     <parameter name="description"><![CDATA[Create frontend/src/utils/equipmentSlotLabels.ts shared module]]></parameter> 
     <parameter name="prompt"><![CDATA[Create shared equipment slot labels module using EquipmentConfig.tsx's most complete version as base. Update CharacterDetail.tsx, Dashboard.tsx, Inventory.tsx, EquipmentConfig.tsx to use it.]]></parameter> 
     <parameter name="subagent_type"><![CDATA[general-purpose]]></parameter> 
   </invoke> 
   <invoke name="TaskCreate"> 
     <parameter name="description"><![CDATA[Extract renderEquipmentSlot to shared component]]></parameter> 
     <parameter name="prompt"><![CDATA[Create EquipmentSlotCell.tsx component from the renderEquipmentSlot() functions in Inventory.tsx and Dashboard.tsx. Update both pages to use the new component.]]></parameter> 
     <parameter name="subagent_type"><![CDATA[general-purpose]]></parameter> 
   </invoke> 
   <invoke name="TaskCreate"> 
     <parameter name="description"><![CDATA[Unify rarity config references across frontend]]></parameter> 
     <parameter name="prompt"><![CDATA[Unify rarity config: add getRarityColor() to ItemTooltip/config/rarityConfig.ts if missing, then update EquipmentConfig.tsx, Collection.tsx, MonsterEncyclopedia.tsx, ItemFilter.tsx, Inventory.tsx, EquipmentSlot.tsx, SkillTag.tsx to import from the shared source instead of local definitions.]]></parameter> 
     <parameter name="subagent_type"><![CDATA[general-purpose]]></parameter> 
   </invoke> 
 </tool_calls>`

	result := toolcall.ParseToolCallsDetailed(text, []string{"TaskCreate"})
	fmt.Printf("SawToolCallSyntax: %v\n", result.SawToolCallSyntax)
	fmt.Printf("Calls: %d\n", len(result.Calls))
	for i, call := range result.Calls {
		fmt.Printf("Call %d: name=%s\n", i+1, call.Name)
		for k, v := range call.Input {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}
