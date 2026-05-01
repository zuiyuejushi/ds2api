package toolcall

import (
	"encoding/json"
	"testing"
)

func TestDSMLInsertCommand(t *testing.T) {
	input := "<｜｜DSML｜｜tool_calls> <｜｜DSML｜｜invoke name=\"bash\"> <｜｜DSML｜｜parameter name=\"command\" string=\"true\">npx prisma db execute --stdin --file /dev/stdin <<< \"INSERT INTO item_templates (id, name, type, subType, rarity, levelReq, dropLevel, baseDefense, baseDamage, baseDamageMax, maxSockets, strReq, dexReq, affixes, fixedAffixes, description, isSuperior, isEthereal, createdAt) VALUES ('rune-ice-peak-01', '冰峰符文 (Ice Peak)', 'rune', NULL, 'unique', 30, 30, 0, 0, 0, 0, 0, 0, '[{\\\"stat\\\":\\\"cold_min\\\",\\\"value\\\":7},{\\\"stat\\\":\\\"cold_max\\\",\\\"value\\\":9},{\\\"stat\\\":\\\"splash\\\",\\\"value\\\":4}]', '[]', '+7-9冰冷伤害，+4%溅射效果', 0, 0, datetime('now'));\"</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"description\" string=\"true\">Insert Ice Peak Rune into item_templates</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"timeout\" string=\"false\">15000</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"workdir\" string=\"true\">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> <｜｜DSML｜｜invoke name=\"bash\"> <｜｜DSML｜｜parameter name=\"command\" string=\"true\">npx prisma db execute --stdin --file /dev/stdin <<< \"INSERT INTO skill_templates (id, name, description, level, damageType, damageMin, damageMax, manaCost, cooldown, effectType, details, tags, createdAt) VALUES ('skill-ice-peak-01', '冰峰符文', '+7-9冰冷伤害，+4%溅射效果', 1, 'cold', 7, 9, 0, 0, 'passive', '{\\\"splash_percent\\\":4,\\\"effectType\\\":\\\"rune_special\\\"}', '[\\\"符文\\\",\\\"冰冷\\\",\\\"溅射\\\"]', datetime('now'));\"</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"description\" string=\"true\">Insert Ice Peak Rune skill into skill_templates</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"timeout\" string=\"false\">15000</｜｜DSML｜｜parameter> <｜｜DSML｜｜parameter name=\"workdir\" string=\"true\">/root/.local/share/opencode/worktree/bf1adfaeee2d2aec4493a0ef0ce3f71f5b368462/glowing-forest/backend</｜｜DSML｜｜parameter> </｜｜DSML｜｜invoke> </｜｜DSML｜｜tool_calls>"

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
