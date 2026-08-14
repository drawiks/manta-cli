package stats

import (
	"strings"
	"testing"

	"github.com/dotabuff/manta/dota"
)

func TestInfKeyTrapCollapse(t *testing.T) {
	if got := infKey(disableTrap, "modifier_mars_arena_of_blood_leash"); got != "mars_arena_of_blood" {
		t.Errorf("leash suffix: got %q", got)
	}
	if got := infKey(disableTrap, "modifier_mars_arena_of_blood_spear"); got != "mars_arena_of_blood" {
		t.Errorf("spear suffix: got %q", got)
	}
	if got := infKey(disableSilence, "modifier_night_stalker_crippling_fear_aura"); got != "night_stalker_crippling_fear" {
		t.Errorf("silence aura: got %q", got)
	}
}

func TestSupplementalDisableFearScaled(t *testing.T) {
	var d disableTrack
	inf := "modifier_nevermore_requiem_fear"
	for i := 0; i < 5; i++ {
		d.add(disableFear, 1, inf, uint32(10+i), 0.4)
		for j := 0; j < 3; j++ {
			d.add(disableFear, 1, inf, uint32(10+i), 0)
		}
	}
	for i := 0; i < 5; i++ {
		for j := 0; j < 4; j++ {
			d.remove(disableFear, 1, inf, uint32(10+i), 0.5)
		}
	}
	got := d.supplementalDisable()
	if got < 1.9 || got > 2.1 {
		t.Errorf("scaled fear supplemental = %v, want ~2", got)
	}
}

func TestSupplementalDisableCripplingFearSilence(t *testing.T) {
	var d disableTrack
	inf := "modifier_night_stalker_crippling_fear"
	for i := 0; i < 10; i++ {
		d.add(disableSilence, 1, inf, uint32(10+i), 0)
		d.remove(disableSilence, 1, inf, uint32(10+i), 5.0)
	}
	if got := d.supplementalDisable(); got != 50 {
		t.Errorf("crippling fear paired silence = %v, want 50", got)
	}
}

func TestSupplementalRequiresAddStunCredit(t *testing.T) {
	var d disableTrack
	inf := "modifier_venomancer_venomous_gale"
	for i := 0; i < 5; i++ {
		d.add(disableRoot, 1, inf, uint32(10+i), 0)
		d.remove(disableRoot, 1, inf, uint32(10+i), 10.0)
	}
	if got := d.supplementalDisable(); got != 0 {
		t.Errorf("root without ADD stun credit = %v, want 0", got)
	}
}

func TestSelfTargetNotTracked(t *testing.T) {
	a := newAggregator()
	a.lookup = func(table string, index int32) (string, bool) {
		switch index {
		case 1:
			return "modifier_axe_berserkers_call_armor", true
		case 2:
			return "npc_dota_hero_axe", true
		}
		return "", false
	}
	bind(a, 0, 100)

	add := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD)
	add.AttackerName = u32p(100)
	add.InflictorName = u32p(1)
	add.TargetName = u32p(100)
	add.IsTargetHero = boolp(true)
	a.handleCombatLog(add, 0)

	rem := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE)
	rem.AttackerName = u32p(100)
	rem.InflictorName = u32p(1)
	rem.TargetName = u32p(100)
	rem.ModifierElapsedDuration = f32p(3.0)
	a.handleCombatLog(rem, 0)

	if got := a.xpStats(100).disable.supplementalDisable(); got != 0 {
		t.Errorf("self-target supplemental = %v, want 0", got)
	}
}

func TestBuildStunSourcesMergedFear(t *testing.T) {
	src := buildStunSources(stunBuildInput{
		stunByInf: map[string]float64{"nevermore_requiem_fear": 7.8},
		disable: disableTrack{
			byInf: map[string]*infTrack{
				"nevermore_requiem_fear": {
					cat: disableFear, addStun: 4.2, removeEl: 11.0,
					addN: 3, addStunN: 3, removeN: 3,
				},
			},
		},
	})
	if len(src) != 1 {
		t.Fatalf("len=%d want 1 merged fear source", len(src))
	}
	if src[0].Category != "fear" || src[0].Inflictor != "nevermore_requiem_fear" {
		t.Errorf("source = %+v", src[0])
	}
	want := 7.8 + (11.0 - 4.2)
	if src[0].Seconds < want-0.01 || src[0].Seconds > want+0.01 {
		t.Errorf("seconds = %v want ~%v", src[0].Seconds, want)
	}
}

func TestClassifyDisable(t *testing.T) {
	cases := map[string]disableCat{
		"modifier_nevermore_requiem_fear":              disableFear,
		"modifier_nevermore_requiem_slow":              disableNone,
		"modifier_naga_siren_song_of_the_siren":        disableSleep,
		"modifier_naga_siren_ensnare":                    disableRoot,
		"modifier_lion_voodoo":                         disableNone,
		"modifier_stunned":                               disableNone,
		"modifier_ghost_state":                           disableNone,
		"modifier_venomancer_venomous_gale":              disableRoot,
		"modifier_axe_berserkers_call":                 disableTaunt,
		"modifier_silencer_global_silence":             disableSilence,
		"modifier_silence":                               disableNone,
		"modifier_silver_edge_break":                     disableBreak,
		"modifier_break":                                 disableNone,
		"modifier_heavens_halberd_debuff":                disableMute,
		"modifier_eul_cyclone":                           disableBanish,
		"modifier_puck_dream_coil":                       disableLeash,
		"modifier_mars_arena_of_blood":                   disableTrap,
		"modifier_mars_arena_of_blood_leash":             disableTrap,
		"modifier_marci_unleash":                         disableNone,
		"modifier_night_stalker_crippling_fear":          disableSilence,
		"modifier_night_stalker_crippling_fear_aura":     disableSilence,
		"modifier_axe_berserkers_call_armor":             disableNone,
		"modifier_drow_ranger_wave_of_silence_buff":      disableNone,
	}
	for inf, want := range cases {
		if got := classifyDisable(inf); got != want {
			t.Errorf("classifyDisable(%q) = %v, want %v", inf, got, want)
		}
	}
}

func TestSupplementalDisableFearSum(t *testing.T) {
	var d disableTrack
	d.add(disableFear, 1, "modifier_nevermore_requiem_fear", 10, 7.8)
	d.remove(disableFear, 1, "modifier_nevermore_requiem_fear", 10, 20.767)
	if got := d.supplementalDisable(); got < 12.9 || got > 13.1 {
		t.Errorf("fear supplemental = %v, want ~13", got)
	}
}

func TestSupplementalDisableRootNoDoubleCount(t *testing.T) {
	var d disableTrack
	d.add(disableRoot, 1, "modifier_naga_siren_ensnare", 10, 84.0)
	d.remove(disableRoot, 1, "modifier_naga_siren_ensnare", 10, 72.867)
	if got := d.supplementalDisable(); got != 0 {
		t.Errorf("root supplemental = %v, want 0 when remove < add stun", got)
	}
}

func TestSupplementalDisableSleepScaled(t *testing.T) {
	var d disableTrack
	inf := "modifier_naga_siren_song_of_the_siren"
	for i := 0; i < 12; i++ {
		d.add(disableSleep, 6, inf, uint32(10+i), 4.8/12)
	}
	for i := 0; i < 53; i++ {
		tgt := uint32(10 + i%12)
		d.remove(disableSleep, 6, inf, tgt, 254.067/53)
	}
	got := d.supplementalDisable()
	if got < 51 || got > 58 {
		t.Errorf("scaled sleep supplemental = %v, want ~52-56", got)
	}
}

func TestBuildStunSourcesScoreboard(t *testing.T) {
	// A present scoreboard m_fStuns is authoritative: no supplemental ADD→REMOVE
	// credit is layered on top, so the source list is the lump scoreboard sum.
	src := buildStunSources(stunBuildInput{
		base:       77.734,
		scoreboard: true,
		disable: disableTrack{
			byInf: map[string]*infTrack{
				"naga_siren_song_of_the_siren": {
					cat: disableSleep, addStun: 4.8, removeEl: 168,
					addN: 31, addStunN: 12, removeN: 31,
				},
			},
		},
	})
	if len(src) != 1 {
		t.Fatalf("len=%d want 1 scoreboard lump source", len(src))
	}
	if src[0].Inflictor != "scoreboard" || src[0].Category != "stun" || src[0].Seconds < 77.7 || src[0].Seconds > 77.8 {
		t.Errorf("source = %+v", src[0])
	}
}

func TestBuildStunSourcesScoreboardZero(t *testing.T) {
	if src := buildStunSources(stunBuildInput{base: 0, scoreboard: true}); src != nil {
		t.Errorf("zero scoreboard stun = %+v, want nil", src)
	}
}

func TestHandleCombatLogSupplementalFear(t *testing.T) {
	a := newAggregator()
	a.lookup = func(table string, index int32) (string, bool) {
		if table == "CombatLogNames" && index == 1 {
			return "modifier_nevermore_requiem_fear", true
		}
		if table == "CombatLogNames" && index == 2 {
			return "npc_dota_hero_pudge", true
		}
		return "", false
	}
	bind(a, 0, 100)

	add := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD)
	add.AttackerName = u32p(100)
	add.InflictorName = u32p(1)
	add.TargetName = u32p(2)
	add.IsTargetHero = boolp(true)
	add.StunDuration = f32p(0.6)
	a.handleCombatLog(add, 0)

	rem := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE)
	rem.AttackerName = u32p(100)
	rem.InflictorName = u32p(1)
	rem.TargetName = u32p(2)
	rem.ModifierElapsedDuration = f32p(4.2)
	a.handleCombatLog(rem, 0)

	total := totalStunDuration(0, a.xpStats(100).disable)
	if total < 3.59 || total > 3.61 {
		t.Errorf("total = %v, want ~3.6", total)
	}
	src := buildStunSources(stunBuildInput{disable: a.xpStats(100).disable})
	if len(src) != 1 || src[0].Category != "fear" {
		t.Errorf("sources = %+v", src)
	}
}

func TestInfKeyDoesNotMatchUnleashAsLeash(t *testing.T) {
	if got := classifyDisable("modifier_marci_unleash"); got != disableNone {
		t.Errorf("unleash = %v, want none", got)
	}
	if strings.Contains("unleash", "leash") && hasLeash("unleash") {
		t.Error("hasLeash should not match unleash substring")
	}
}
