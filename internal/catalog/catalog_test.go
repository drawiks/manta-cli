package catalog

import "testing"

func TestClassifyModifier(t *testing.T) {
	cases := []struct {
		inf      string
		kind     ModKind
		category string
		clean    string
	}{
		{"modifier_dazzle_shallow_grave", KindBuff, "save", "dazzle_shallow_grave"},
		{"modifier_axe_berserkers_call", KindTaunt, "", "axe_berserkers_call"},
		{"modifier_heavens_halberd", KindDisarm, "", "heavens_halberd"},
		{"modifier_item_urn_heal", KindHeal, "", "item_urn_heal"},
		{"modifier_silver_edge_debuff", KindBreak, "", "silver_edge"},
		{"modifier_naga_siren_ensnare", KindRoot, "", "naga_siren_ensnare"},
	}
	for _, c := range cases {
		got := ClassifyModifier(c.inf)
		if got.Kind != c.kind || got.Category != c.category || got.Clean != c.clean {
			t.Errorf("%s: got kind=%v cat=%q clean=%q, want kind=%v cat=%q clean=%q",
				c.inf, got.Kind, got.Category, got.Clean, c.kind, c.category, c.clean)
		}
	}
}
