package toolcall

import (
	"encoding/json"
	"testing"
)

func TestDSMLComplexBashCommand(t *testing.T) {
	input := "<｜｜DSML｜｜tool_calls> <｜｜DSML｜｜invoke name=\"bash\"> <｜｜DSML｜｜parameter name=\"command\" string=\"true\">npx ts-node --transpile-only --compiler-options '{\"module\":\"commonjs\"}' -e \"const { PrismaClient } = require('@prisma/client'); const p = new PrismaClient(); async function main() { const items = await p.itemTemplate.findMany({ where: { type: 'rune' }, select: { id: true, name: true, type: true, rarity: true, levelReq: true, affixes: true, description: true } }); console.log(JSON.stringify(items, null, 2)); } main().then(() => p'\\`$disconnect')\"</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"description\" string=\"true\">Query existing rune-type item_templates</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"timeout\" string=\"false\">15000</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"workdir\" string=\"true\">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> <｜｜DSML｜｜invoke name=\"bash\"> <｜｜DSML｜｜parameter name=\"command\" string=\"true\">npx ts-node --transpile-only --compiler-options '{\"module\":\"commonjs\"}' -e \"const { PrismaClient } = require('@prisma/client'); const p = new PrismaClient(); async function main() { const skill = await p.skillTemplate.findFirst({ select: { id: true, name: true, level: true, effectType: true, damageType: true, damageMin: true, damageMax: true, details: true, description: true, manaCost: true, cooldown: true, tags: true } }); console.log(JSON.stringify(skill, null, 2)); } main().then(() => p'\\`$disconnect')\"</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"description\" string=\"true\">Sample skill_template for format reference</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"timeout\" string=\"false\">15000</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"workdir\" string=\"true\">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> </｜｜DSML｜｜tool_calls>"

	result := ParseToolCalls(input, nil)

	if len(result) == 0 {
		t.Fatalf("Expected tool calls to be parsed, got none")
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(result))
	}

	for i, tc := range result {
		if tc.Name != "bash" {
			t.Errorf("Call %d: expected name 'bash', got '%s'", i, tc.Name)
		}
		if tc.Input == nil {
			t.Errorf("Call %d: expected non-nil input", i)
			continue
		}

		// Check command parameter exists
		if _, ok := tc.Input["command"]; !ok {
			t.Errorf("Call %d: missing command parameter", i)
		}

		// Check timeout is parsed as number
		timeout, ok := tc.Input["timeout"]
		if !ok {
			t.Errorf("Call %d: missing timeout parameter", i)
		} else {
			// Should be float64 (number)
			if _, isNum := timeout.(float64); !isNum {
				t.Errorf("Call %d: timeout should be float64, got %T", i, timeout)
			}
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Parsed result:\n%s", string(data))
}
