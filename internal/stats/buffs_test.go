package stats

import (
	"strings"
	"testing"

	"github.com/dotabuff/manta/dota"
)

func TestClassifyBuff(t *testing.T) {
	cases := map[string]struct {
		cat  buffCat
		name string
	}{
		"modifier_dazzle_shallow_grave":             {buffSave, "dazzle_shallow_grave"},
		"modifier_oracle_false_promise":             {buffSave, "oracle_false_promise"},
		"modifier_winter_wyvern_cold_embrace":       {buffSave, "winter_wyvern_cold_embrace"},
		"modifier_item_forcestaff_active":           {buffSave, "item_force_staff"},
		"modifier_item_cyclone":                     {buffSave, "item_cyclone"},
		"modifier_omniknight_guardian_angel":        {buffSave, "omniknight_guardian_angel"},
		"modifier_omniknight_purification":          {buffPurge, "omniknight_purification"},
		"modifier_item_lotus_orb_active":            {buffPurge, "item_lotus_orb"},
		"modifier_item_guardian_greaves":            {buffPurge, "item_guardian_greaves"},
		"modifier_abaddon_aphotic_shield":           {buffShield, "abaddon_aphotic_shield"},
		"modifier_item_pipe_barrier":                {buffShield, "item_pipe_of_insight"},
		"modifier_item_pavise_shield":               {buffShield, "item_pavise_shield"},
		"modifier_omniknight_repel":                 {buffSpellImmunity, "omniknight_repel"},
		"modifier_item_glimmer_cape_fade":           {buffInvisibility, "item_glimmer_cape"},
		"modifier_smoke_of_deceit":                  {buffInvisibility, "smoke_of_deceit"},
		"modifier_mirana_moonlight_shadow":          {buffInvisibility, "mirana_moonlight_shadow"},
		"modifier_item_solar_crest_armor_addition":  {buffStats, "item_solar_crest"},
		"modifier_ogre_magi_bloodlust":              {buffStats, "ogre_magi_bloodlust"},
		"modifier_item_drum_of_endurance_active":    {buffHaste, "item_drum_of_endurance"},
		"modifier_keeper_of_the_light_chakra_magic": {buffManaRestore, "keeper_of_the_light_chakra_magic"},
		"modifier_axe_berserkers_call_armor":        {buffNone, ""},
		"modifier_nevermore_requiem_fear":           {buffNone, ""},
	}
	for inf, want := range cases {
		cat, name := classifyBuff(inf)
		if cat != want.cat || name != want.name {
			t.Errorf("classifyBuff(%q) = (%v,%q), want (%v,%q)", inf, cat, name, want.cat, want.name)
		}
	}
}

// testBuffAggregator builds an aggregator resolving CombatLogNames indexes.
func testBuffAggregator(names map[int32]string) *aggregator {
	a := newAggregator()
	a.lookup = func(table string, index int32) (string, bool) {
		if table != "CombatLogNames" {
			return "", false
		}
		n, ok := names[index]
		return n, ok
	}
	return a
}

func TestBuffPairedUptime(t *testing.T) {
	names := map[int32]string{
		1: "modifier_dazzle_shallow_grave",
		2: "modifier_item_glimmer_cape_fade",
	}
	a := testBuffAggregator(names)
	bind(a, 0, 100)

	mod := func(typ dota.DOTA_COMBATLOG_TYPES, inf, att, tgt uint32, elapsed float32, hero bool) *dota.CMsgDOTACombatLogEntry {
		e := entry(typ)
		e.InflictorName = u32p(inf)
		e.AttackerName = u32p(att)
		e.TargetName = u32p(tgt)
		e.AttackerTeam = u32p(teamRadiant)
		e.TargetTeam = u32p(teamRadiant)
		e.IsTargetHero = boolp(hero)
		e.ModifierElapsedDuration = f32p(elapsed)
		return e
	}

	// Shallow grave on ally 200 for 5.0 s.
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD, 1, 100, 200, 0, true), 0)
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE, 1, 100, 200, 5.0, true), 0)
	// Glimmer on ally 201 twice (2.5 + 3.0).
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD, 2, 100, 201, 0, true), 0)
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE, 2, 100, 201, 2.5, true), 0)
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD, 2, 100, 202, 0, true), 0)
	a.handleCombatLog(mod(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE, 2, 100, 202, 3.0, true), 0)

	b := a.xpStats(100).buff
	grave := b.byInf["dazzle_shallow_grave"]
	if grave == nil || grave.uptime != 5.0 || grave.count != 1 {
		t.Fatalf("grave = %+v, want uptime 5 count 1", grave)
	}
	glim := b.byInf["item_glimmer_cape"]
	if glim == nil || glim.uptime != 5.5 || glim.count != 2 {
		t.Fatalf("glimmer = %+v, want uptime 5.5 count 2", glim)
	}
}

func TestBuffFilters(t *testing.T) {
	names := map[int32]string{
		1: "modifier_dazzle_shallow_grave",
		2: "modifier_lina_fiery_soul",
	}
	a := testBuffAggregator(names)
	bind(a, 0, 100)

	add := func(mut func(*dota.CMsgDOTACombatLogEntry)) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD)
		e.InflictorName = u32p(1)
		e.AttackerName = u32p(100)
		e.TargetName = u32p(200)
		e.AttackerTeam = u32p(teamRadiant)
		e.TargetTeam = u32p(teamRadiant)
		e.IsTargetHero = boolp(true)
		mut(e)
		a.handleCombatLog(e, 0)
	}

	// All excluded events must leave the tracker empty.
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.AuraModifier = boolp(true) })     // aura
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.HiddenModifier = boolp(true) })   // hidden
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.TargetTeam = u32p(teamDire) })    // enemy target
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.TargetTeam = u32p(0) })           // missing teams
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.IsTargetHero = boolp(false) })    // not a hero target
	add(func(e *dota.CMsgDOTACombatLogEntry) { e.IsTargetIllusion = boolp(true) }) // illusion

	// Self-only buffs are not in the catalog: even cast on self, excluded.
	self := func(e *dota.CMsgDOTACombatLogEntry) {
		e.InflictorName = u32p(2)
		e.TargetName = u32p(100)
		e.TargetIsSelf = boolp(true)
	}
	add(self)

	if got := a.xpStats(100).buff.totalUptime(); got != 0 {
		t.Errorf("excluded events tracked uptime %v, want 0", got)
	}
}

func TestSelfCastBuffCounted(t *testing.T) {
	names := map[int32]string{1: "modifier_dazzle_shallow_grave"}
	a := testBuffAggregator(names)
	bind(a, 0, 100)

	// Dazzle casts shallow grave on himself: a real buff (ally-capable ability).
	add := func(elapsed float32) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD)
		e.InflictorName = u32p(1)
		e.AttackerName = u32p(100)
		e.TargetName = u32p(100)
		e.AttackerTeam = u32p(teamRadiant)
		e.TargetTeam = u32p(teamRadiant)
		e.IsTargetHero = boolp(true)
		e.TargetIsSelf = boolp(true)
		a.handleCombatLog(e, 0)
		rm := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE)
		rm.InflictorName = u32p(1)
		rm.AttackerName = u32p(100)
		rm.TargetName = u32p(100)
		rm.AttackerTeam = u32p(teamRadiant)
		rm.TargetTeam = u32p(teamRadiant)
		rm.IsTargetHero = boolp(true)
		rm.TargetIsSelf = boolp(true)
		rm.ModifierElapsedDuration = f32p(elapsed)
		a.handleCombatLog(rm, 0)
	}
	add(5.0)
	add(3.0)

	x := a.xpStats(100)
	if x.buff.totalUptime() != 8.0 {
		t.Errorf("self-cast uptime = %v, want 8.0", x.buff.totalUptime())
	}
	if got := buildBuffSources(x); len(got) != 1 || got[0].Value != 8.0 || got[0].Count != 2 {
		t.Errorf("buff sources = %+v, want one save with value 8 count 2", got)
	}
}

// TestBuffRemoveWithoutAddIgnored verifies the pending-pair guard: a REMOVE
// with no matching ADD never credits uptime.
func TestBuffRemoveWithoutAddIgnored(t *testing.T) {
	a := testBuffAggregator(map[int32]string{1: "modifier_dazzle_shallow_grave"})
	bind(a, 0, 100)
	e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE)
	e.InflictorName = u32p(1)
	e.AttackerName = u32p(100)
	e.TargetName = u32p(200)
	e.AttackerTeam = u32p(teamRadiant)
	e.TargetTeam = u32p(teamRadiant)
	e.IsTargetHero = boolp(true)
	e.ModifierElapsedDuration = f32p(5.0)
	a.handleCombatLog(e, 0)
	if got := a.xpStats(100).buff.totalUptime(); got != 0 {
		t.Errorf("orphan REMOVE credited %v, want 0", got)
	}
}

func TestAllyHealCredits(t *testing.T) {
	names := map[int32]string{
		1: "dazzle_shadow_wave",
		2: "item_urn_of_shadows",
	}
	a := testBuffAggregator(names)
	bind(a, 0, 100)

	heal := func(inf, att, tgt uint32, v uint32, mut func(*dota.CMsgDOTACombatLogEntry)) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL)
		e.InflictorName = u32p(inf)
		e.AttackerName = u32p(att)
		e.TargetName = u32p(tgt)
		e.AttackerTeam = u32p(teamRadiant)
		e.TargetTeam = u32p(teamRadiant)
		e.IsTargetHero = boolp(true)
		e.Value = u32p(v)
		mut(e)
		a.handleCombatLog(e, 0)
	}

	heal(1, 100, 200, 400, func(*dota.CMsgDOTACombatLogEntry) {}) // shadow wave on ally
	heal(2, 100, 201, 250, func(*dota.CMsgDOTACombatLogEntry) {}) // urn on ally
	heal(1, 100, 200, 99, func(e *dota.CMsgDOTACombatLogEntry) { e.HealFromLifesteal = boolp(true) })
	heal(1, 100, 200, 88, func(e *dota.CMsgDOTACombatLogEntry) { e.HealFromRegen = boolp(true) })
	heal(1, 100, 100, 500, func(*dota.CMsgDOTACombatLogEntry) {})                                  // self-heal
	heal(1, 100, 300, 700, func(e *dota.CMsgDOTACombatLogEntry) { e.TargetTeam = u32p(teamDire) }) // enemy

	x := a.xpStats(100)
	// Self-heal is a real buff too (self-cast of an ally-capable ability).
	if x.buffHeal != 1150 {
		t.Errorf("buffHeal = %v, want 1150", x.buffHeal)
	}
	if x.buffHealN != 3 {
		t.Errorf("buffHealN = %d, want 3", x.buffHealN)
	}
	if x.buffHealInf["dazzle_shadow_wave"] != 900 || x.buffHealInfN["dazzle_shadow_wave"] != 2 {
		t.Errorf("shadow wave = %v/%d", x.buffHealInf["dazzle_shadow_wave"], x.buffHealInfN["dazzle_shadow_wave"])
	}
	if x.buffHealInf["item_urn_of_shadows"] != 250 {
		t.Errorf("urn = %v, want 250", x.buffHealInf["item_urn_of_shadows"])
	}
}

func TestAllyHealNoLookupFallsBack(t *testing.T) {
	a := newAggregator() // lookup == nil
	bind(a, 0, 100)
	e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL)
	e.AttackerName = u32p(100)
	e.TargetName = u32p(200)
	e.AttackerTeam = u32p(teamRadiant)
	e.TargetTeam = u32p(teamRadiant)
	e.IsTargetHero = boolp(true)
	e.Value = u32p(300)
	a.handleCombatLog(e, 0)
	x := a.xpStats(100)
	if x.buffHeal != 300 || x.buffHealInf["heal"] != 300 {
		t.Errorf("no-lookup heal = %v / %v", x.buffHeal, x.buffHealInf)
	}
}

func TestBuildBuffSources(t *testing.T) {
	var x xpStats
	x.buff.byInf = map[string]*buffInf{
		"item_glimmer_cape":    {cat: buffInvisibility, uptime: 4.0, count: 2},
		"dazzle_shallow_grave": {cat: buffSave, uptime: 5.0, count: 1},
	}
	x.buffHealInf = map[string]float64{"dazzle_shadow_wave": 1200.0}
	x.buffHealInfN = map[string]int{"dazzle_shadow_wave": 3}

	src := buildBuffSources(x)
	if len(src) != 3 {
		t.Fatalf("len = %d, want 3", len(src))
	}
	// Sorted by value descending: heal 1200, grave 5, glimmer 4.
	if src[0].Category != "heal" || src[0].Value != 1200 || src[0].Count != 3 {
		t.Errorf("first source = %+v", src[0])
	}
	if src[1].Inflictor != "dazzle_shallow_grave" || src[1].Category != "save" || src[1].Count != 1 {
		t.Errorf("second source = %+v", src[1])
	}
	if src[2].Inflictor != "item_glimmer_cape" || src[2].Category != "invisibility" {
		t.Errorf("third source = %+v", src[2])
	}
	if buildBuffSources(xpStats{}) != nil {
		t.Error("empty xpStats should produce nil sources")
	}
}

func TestBuffGoldenUnaffected(t *testing.T) {
	// The golden fixture feeds events without team fields; the strict
	// same-team filter must keep buff_sources empty there.
	a := newAggregator()
	e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL)
	e.AttackerName = u32p(100)
	e.IsTargetHero = boolp(true)
	e.Value = u32p(200)
	a.handleCombatLog(e, 0)
	if got := buildBuffSources(a.xpStats(100)); got != nil {
		t.Errorf("buff sources without teams = %+v, want nil", got)
	}
}

func TestBuffCatName(t *testing.T) {
	if buffCatName(buffSave) != "save" || buffCatName(buffManaRestore) != "mana_restore" || buffCatName(buffNone) != "other" {
		t.Error("bad category names")
	}
	if strings.Contains(buffCatName(buffHaste), " ") {
		t.Errorf("category names must not contain spaces, got %q", buffCatName(buffHaste))
	}
}
