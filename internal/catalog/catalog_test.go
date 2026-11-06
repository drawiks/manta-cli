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
		{"modifier_shadow_demon_disruption", KindBuff, "save", "shadow_demon_disruption", false},
		{"modifier_item_sphere_target", KindBuff, "save", "item_sphere_target", false},
		{"modifier_omniknight_martyr", KindBuff, "spell_immunity", "omniknight_martyr", false},
		{"modifier_item_pipe_barrier", KindBuff, "shield", "item_pipe_barrier", false},
		{"modifier_item_pavise_shield", KindBuff, "shield", "item_pavise_shield", false},
		{"modifier_item_crimson_guard_extra", KindBuff, "shield", "item_crimson_guard_extra", false},
		{"modifier_sven_warcry", KindBuff, "buff_stats", "sven_warcry", false},
		{"modifier_troll_warlord_battle_trance_aura", KindBuff, "buff_stats", "troll_warlord_battle_trance", false},
		{"modifier_item_boots_of_bearing_active", KindBuff, "buff_stats", "item_boots_of_bearing", false},
		{"modifier_item_ancient_janggo_active", KindBuff, "buff_haste", "item_ancient_janggo", false},
		{"modifier_marci_companion_run_ally_movespeed", KindBuff, "buff_haste", "marci_companion_run_ally_movespeed", false},
		{"modifier_grimstroke_spirit_walk_buff", KindBuff, "buff_haste", "grimstroke_spirit_walk", true},
		{"modifier_marci_bodyguarded", KindBuff, "buff_stats", "marci_bodyguarded", false},
		{"modifier_chen_divine_favor_armor_buff", KindBuff, "buff_stats", "chen_divine_favor", false},
		{"modifier_undying_tombstone_bunker", KindNone, "", "undying_tombstone_bunker", false},
		{"modifier_undying_tombstone_bunker_cooldown", KindNone, "", "undying_tombstone_bunker_cooldown", false},
		{"modifier_item_solar_crest_armor_addition", KindNone, "", "item_solar_crest_armor_addition", false},
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
		{"modifier_marci_unleash_pulse_silence", KindSilence, "", "marci_unleash_pulse_silence", false},
		{"modifier_oracle_purifying_flames", KindNone, "", "oracle_purifying_flames", false},
		{"modifier_oracle_fortunes_end_purge", KindRoot, "", "oracle_fortunes_end_purge", false},
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
		"bounty_hunter_wind_walk",
		"marci_bodyguard",
		"item_pipe_of_insight",
		"item_pavise",
		"chen_divine_favor_armor",
		"item_drum",
		"omniknight_repel",
	} {
		if _, ok := BuffCategories[key]; ok {
			t.Errorf("BuffCategories contains removed key %q", key)
		}
	}
	for key, category := range map[string]string{
		"oracle_false_promise_timer":         "save",
		"wind_waker":                         "save",
		"undying_tombstone_bunker_friendly":  "save",
		"shadow_demon_disruption":            "save",
		"item_sphere_target":                 "save",
		"omniknight_martyr":                  "spell_immunity",
		"item_pipe_barrier":                  "shield",
		"item_pavise_shield":                 "shield",
		"item_crimson_guard_extra":           "shield",
		"sven_warcry":                        "buff_stats",
		"troll_warlord_battle_trance_aura":   "buff_stats",
		"item_boots_of_bearing_active":       "buff_stats",
		"item_ancient_janggo_active":         "buff_haste",
		"marci_companion_run_ally_movespeed": "buff_haste",
		"grimstroke_spirit_walk_buff":        "buff_haste",
	} {
		if BuffCategories[key] != category {
			t.Errorf("BuffCategories[%q] = %q, want %q", key, BuffCategories[key], category)
		}
	}
	if !PurgeModifiers["disperser_movespeed_buff"] {
		t.Error("disperser movespeed buff must be a purge")
	}
}
