package catalog

import "testing"

func TestClassifyModifier(t *testing.T) {
	cases := []struct {
		inf      string
		kind     ModKind
		category string
		clean    string
		purge    bool
	}{
		{"modifier_dazzle_shallow_grave", KindBuff, "save", "dazzle_shallow_grave", false},
		{"modifier_oracle_false_promise_timer", KindBuff, "save", "oracle_false_promise", false},
		{"modifier_wind_waker", KindBuff, "save", "wind_waker", false},
		{"modifier_undying_tombstone_bunker_friendly", KindBuff, "save", "undying_tombstone_bunker_friendly", false},
		{"modifier_undying_tombstone_bunker", KindNone, "", "undying_tombstone_bunker", false},
		{"modifier_undying_tombstone_bunker_cooldown", KindNone, "", "undying_tombstone_bunker_cooldown", false},
		{"modifier_disperser_movespeed_buff", KindNone, "", "disperser_movespeed", true},
		{"modifier_eul_cyclone", KindNone, "", "eul_cyclone", false},
		{"modifier_pugna_decrepify", KindNone, "", "pugna_decrepify", false},
		{"modifier_underlord_dark_rift", KindNone, "", "underlord_dark_rift", false},
		{"modifier_abaddon_borrowed_time", KindNone, "", "abaddon_borrowed_time", false},
		{"modifier_axe_berserkers_call", KindTaunt, "", "axe_berserkers_call", false},
		{"modifier_heavens_halberd", KindDisarm, "", "heavens_halberd", false},
		{"modifier_item_urn_heal", KindHeal, "", "item_urn_heal", false},
		{"modifier_silver_edge_debuff", KindBreak, "", "silver_edge", false},
		{"modifier_naga_siren_ensnare", KindRoot, "", "naga_siren_ensnare", false},
	}
	for _, c := range cases {
		got := ClassifyModifier(c.inf)
		if got.Kind != c.kind || got.Category != c.category || got.Clean != c.clean || got.Purge != c.purge {
			t.Errorf("%s: got kind=%v cat=%q clean=%q purge=%v, want kind=%v cat=%q clean=%q purge=%v",
				c.inf, got.Kind, got.Category, got.Clean, got.Purge, c.kind, c.category, c.clean, c.purge)
		}
	}
}

func TestBuffAndPurgeMaps(t *testing.T) {
	for _, key := range []string{
		"underlord_dark_rift",
		"eul_cyclone",
		"pugna_decrepify",
		"abaddon_borrowed_time",
		"disperser_movespeed_buff",
	} {
		if _, ok := BuffCategories[key]; ok {
			t.Errorf("BuffCategories contains removed key %q", key)
		}
	}
	if BuffCategories["oracle_false_promise_timer"] != "save" {
		t.Error("oracle false promise timer must be a save")
	}
	if BuffCategories["wind_waker"] != "save" {
		t.Error("wind waker must be a save")
	}
	if BuffCategories["undying_tombstone_bunker_friendly"] != "save" {
		t.Error("undying tombstone bunker friendly must be a save")
	}
	if !PurgeModifiers["disperser_movespeed_buff"] {
		t.Error("disperser movespeed buff must be a purge")
	}
}
