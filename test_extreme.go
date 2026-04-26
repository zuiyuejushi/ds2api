package main

import (
	"fmt"
	"ds2api/internal/toolcall"
)

func main() {
	// 极端损坏的格式
	text := `Read  parameter> <parameter name="limit"><![CDATA<parameter name="offset">40`

	result := toolcall.ParseToolCallsDetailed(text, []string{"Read"})
	fmt.Printf("Input: %s\n\n", text)
	fmt.Printf("SawToolCallSyntax: %v\n", result.SawToolCallSyntax)
	fmt.Printf("Calls: %d\n", len(result.Calls))

	// 尝试修复
	repaired, ok := toolcall.RepairXMLToolCalls(text)
	fmt.Printf("\nRepair OK: %v\n", ok)
	fmt.Printf("Repaired: %s\n", repaired)
}
