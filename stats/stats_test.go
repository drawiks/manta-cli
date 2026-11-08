package stats

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchemaMatchesOdotaCLI(t *testing.T) {
	res := Match{
		MatchID:     1,
		DurationSec: 2.5,
		Players: []Player{{
			PlayerID:       0,
			Hero:           "rubick",
			Position:       4,
			Lane:           "supp",
			StunSources:    []SourceEntry{{Inflictor: "scoreboard", Category: "stun", Duration: 3}},
			BuffSources:    []SourceEntry{{Inflictor: "item_lotus_orb", Category: "purge", Count: 2}},
			RootSources:    []SourceEntry{{Inflictor: "naga_siren_ensnare", Category: "root", Duration: 4}},
			BuffDuration:   5,
			RootsDuration:  4,
			GoldSpentWards: 50,
			FirstBlood:     true,
		}},
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{
		`"match_id"`, `"duration_sec"`, `"radiant_win"`, `"players"`,
		`"steam_id"`, `"player_id"`, `"hero_id"`, `"hero"`, `"team"`, `"name"`,
		`"position"`, `"lane"`, `"kills"`, `"deaths"`, `"assists"`, `"level"`,
		`"last_hits"`, `"networth"`, `"gpm"`, `"xpm"`, `"healing"`, `"mana_restored"`,
		`"hero_damage"`, `"damage_taken"`, `"tower_damage"`, `"time_dead"`,
		`"stun_duration"`, `"stun_sources"`, `"buff_duration"`, `"buff_sources"`,
		`"fear_duration"`, `"roots_duration"`, `"leash_duration"`, `"trap_duration"`,
		`"taunt_duration"`, `"silence_duration"`, `"break_duration"`, `"disarm_duration"`,
		`"gold_spent_wards"`,
		`"gold_spent_smoke"`, `"gold_spent_dust"`, `"gold_lost"`, `"camps_stacked"`,
		`"creeps_stacked"`, `"rune_pickups"`, `"wisdoms_captured"`,
		`"watchers_captured"`, `"lotuses_gathered"`, `"first_blood"`, `"courier_kills"`,
		`"inflictor"`, `"category"`, `"duration"`, `"count"`,
	} {
		if !strings.Contains(s, key) {
			t.Errorf("JSON is missing %s: %s", key, s)
		}
	}
	for _, stale := range []string{`"buffs_duration"`, `"seconds"`, `"gold_wards"`, `"gold_smoke"`, `"gold_dust"`, `"heal_duration"`, `"heal_value"`, `"heal_sources"`} {
		if strings.Contains(s, stale) {
			t.Errorf("JSON still contains old field %s: %s", stale, s)
		}
	}
}
