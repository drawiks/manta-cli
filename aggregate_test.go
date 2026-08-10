package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dotabuff/manta/dota"
)

// --- helpers to construct synthetic combat-log entries ---

func u32p(v uint32) *uint32 { return &v }
func f32p(v float32) *float32 {
	return &v
}
func boolp(v bool) *bool { return &v }

func clType(t dota.DOTA_COMBATLOG_TYPES) *dota.DOTA_COMBATLOG_TYPES { return &t }

func entry(t dota.DOTA_COMBATLOG_TYPES) *dota.CMsgDOTACombatLogEntry {
	return &dota.CMsgDOTACombatLogEntry{Type: clType(t)}
}

// fakePR is a synthetic prReader for tests.
type fakePR struct {
	slots map[int]prSlot
}

func (f fakePR) slot(i int) (prSlot, bool) {
	s, ok := f.slots[i]
	if !ok || s.steamID == 0 {
		return prSlot{}, false
	}
	return s, true
}

// fakeTeam is a synthetic teamReader for tests.
type fakeTeam struct {
	td map[string]map[int32]teamData
}

func (f fakeTeam) team(side string, teamSlot int32) (teamData, bool) {
	td, ok := f.td[side][teamSlot]
	return td, ok
}

// clEvent pairs a combat-log entry with the tick it arrived on.
type clEvent struct {
	tick uint32
	e    *dota.CMsgDOTACombatLogEntry
}

func ev(tick uint32, e *dota.CMsgDOTACombatLogEntry) clEvent {
	return clEvent{tick: tick, e: e}
}

// feed runs a sequence of combat-log entries through the aggregator.
func feed(a *aggregator, events []clEvent) {
	for _, x := range events {
		a.handleCombatLog(x.e, x.tick)
	}
}

// bind wires PR slot <-> xpID via a PLAYERSTATS entry (spec §6.3).
func bind(a *aggregator, slot, xpID uint32) {
	e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS)
	e.TargetName = u32p(slot)
	e.Value = u32p(xpID)
	a.handleCombatLog(e, 0)
}

func TestAggregateGoldenJSON(t *testing.T) {
	a := newAggregator()

	feed(a, []clEvent{
		ev(0, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE)
			e.Value = u32p(5)
			e.Timestamp = f32p(100.0)
			return e
		}()),
		ev(3000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
			e.AttackerName = u32p(100)
			e.IsTargetBuilding = boolp(true)
			e.Value = u32p(500)
			return e
		}()),
		ev(3000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL)
			e.AttackerName = u32p(100)
			e.IsTargetHero = boolp(true)
			e.Value = u32p(200)
			return e
		}()),
		ev(6000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD)
			e.AttackerName = u32p(100)
			e.StunDuration = f32p(1.5)
			return e
		}()),
		ev(6000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
			e.AttackerName = u32p(100)
			e.IsTargetHero = boolp(true)
			return e
		}()),
		ev(6000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
			e.AttackerName = u32p(101)
			e.IsTargetHero = boolp(true)
			return e
		}()),
		ev(9000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD)
			e.TargetName = u32p(100)
			e.Value = u32p(6000)
			return e
		}()),
		ev(9000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_XP)
			e.TargetName = u32p(100)
			e.Value = u32p(12000)
			return e
		}()),
		ev(9000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS)
			e.TargetName = u32p(0)
			e.Value = u32p(100)
			e.LastHits = u32p(230)
			e.Networth = u32p(18900)
			return e
		}()),
		ev(12000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS)
			e.TargetName = u32p(1)
			e.Value = u32p(101)
			e.LastHits = u32p(150)
			e.Networth = u32p(12000)
			return e
		}()),
		ev(12000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
			e.AttackerName = u32p(101)
			e.IsTargetBuilding = boolp(true)
			e.Value = u32p(700)
			return e
		}()),
		ev(12000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD)
			e.TargetName = u32p(101)
			e.Value = u32p(5000)
			return e
		}()),
		ev(12000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_XP)
			e.TargetName = u32p(101)
			e.Value = u32p(10000)
			return e
		}()),
		ev(15000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_TEAM_BUILDING_KILL)
			e.TargetTeam = u32p(3)
			return e
		}()),
		ev(18000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
			e.AttackerName = u32p(100)
			e.IsTargetBuilding = boolp(true)
			e.Value = u32p(300)
			return e
		}()),
		ev(18000, func() *dota.CMsgDOTACombatLogEntry {
			e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE)
			e.Value = u32p(6)
			e.Timestamp = f32p(600.0)
			return e
		}()),
	})

	pr := fakePR{slots: map[int]prSlot{
		0: {
			steamID: 100, team: teamRadiant, teamSlot: 0, name: "Pudge",
			kills: 8, deaths: 3, assists: 12, level: 25,
			bountyRunes: 1, powerRunes: 2, waterRunes: 0,
			heroID: 14, heroHandle: 10,
		},
		1: {
			steamID: 200, team: teamDire, teamSlot: 1, name: "Slark",
			kills: 5, deaths: 6, assists: 7, level: 21,
			heroID: 93, heroHandle: 11,
		},
	}}

	teams := fakeTeam{td: map[string]map[int32]teamData{
		"radiant": {0: {campsStacked: 4, creepsStacked: 9}},
		"dire":    {1: {campsStacked: 2, creepsStacked: 5}},
	}}

	heroes := map[uint64]heroInfo{
		10: {name: "Pudge", playerID: 0},
		11: {name: "Slark", playerID: 2},
	}

	res := buildResult(a, pr, heroes, teams, 7047839963)

	got, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	golden, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != string(golden) {
		t.Errorf("JSON mismatch\n--- got ---\n%s\n--- want ---\n%s", got, golden)
	}

	if got := a.durationSec(); got != 500 {
		t.Errorf("durationSec = %d, want 500", got)
	}
	if !a.radiantWin() {
		t.Error("radiantWin = false, want true")
	}
}

func TestAggregateEmptyResult(t *testing.T) {
	a := newAggregator()
	res := buildResult(a, fakePR{}, nil, nil, 0)
	if len(res.Players) != 0 {
		t.Fatalf("expected no players, got %d", len(res.Players))
	}
	if res.DurationSec != 0 {
		t.Errorf("durationSec = %d, want 0", res.DurationSec)
	}
	if res.RadiantWin {
		t.Error("radiantWin should be false")
	}
	if res.MatchID != 0 {
		t.Errorf("match_id = %d, want 0", res.MatchID)
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{\n  \"duration_sec\": 0,\n  \"radiant_win\": false,\n  \"players\": []\n}" {
		t.Errorf("unexpected empty JSON:\n%s", b)
	}
}

func TestDurationFromGameStates(t *testing.T) {
	a := newAggregator()

	// GAME_IN_PROGRESS (5) with its game-clock timestamp; no duration yet.
	gs := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE)
	gs.Value = u32p(5)
	gs.Timestamp = f32p(227.8667)
	a.handleCombatLog(gs, 6011)

	tbk := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_TEAM_BUILDING_KILL)
	tbk.TargetTeam = u32p(2)
	a.handleCombatLog(tbk, 30000)

	if got := a.durationSec(); got != 0 {
		t.Errorf("durationSec = %d, want 0 while game in progress", got)
	}
	if a.radiantWin() {
		t.Error("radiantWin = true, want false")
	}

	// POST_GAME (6): duration = POST_GAME ts - GAME_IN_PROGRESS ts, truncated.
	post := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE)
	post.Value = u32p(6)
	post.Timestamp = f32p(2691.4001)
	a.handleCombatLog(post, 80097)

	if got := a.durationSec(); got != 2463 {
		t.Errorf("durationSec = %d, want 2463", got)
	}
}

func TestNilFieldsDoNotPanic(t *testing.T) {
	a := newAggregator()
	for _, typ := range []dota.DOTA_COMBATLOG_TYPES{
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_FIRST_BLOOD,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_XP,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_ITEM,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PURCHASE,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS,
		dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_TEAM_BUILDING_KILL,
	} {
		a.handleCombatLog(entry(typ), 42)
	}
}

func TestBuildPlayerRunePickupsAndMissingHero(t *testing.T) {
	slot := prSlot{
		steamID: 300, team: teamDire, name: "no-hero",
		bountyRunes: 1, powerRunes: 0, waterRunes: 2,
		heroID: 1, heroHandle: 9999,
	}
	a := newAggregator()
	p := buildPlayer(a, slot, nil, nil, 3, 300)
	if p.PlayerID != 6 { // 2*slotIdx fallback
		t.Errorf("playerID = %d, want 6", p.PlayerID)
	}
	if p.RunePickups != 3 {
		t.Errorf("rune_pickups = %d, want 3", p.RunePickups)
	}
	if p.Hero != "" {
		t.Errorf("hero = %q, want empty", p.Hero)
	}
	if p.Team != "dire" {
		t.Errorf("team = %q, want dire", p.Team)
	}
}

// TestFirstBloodFromEventType18 checks that the FIRST_BLOOD (type 18) event
// pins the moment of the first kill on the map: even when a hero DEATH
// arrives earlier (e.g. a pre-horn fountain kill), the credit goes to the
// hero death at/after the type-18 tick.
func TestFirstBloodFromEventType18(t *testing.T) {
	a := newAggregator()

	// Early hero death (not first blood) at tick 100.
	d1 := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
	d1.AttackerName = u32p(200)
	d1.IsTargetHero = boolp(true)
	a.handleCombatLog(d1, 100)

	// FIRST_BLOOD fires at tick 500 (no attacker carried).
	fb := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_FIRST_BLOOD)
	a.handleCombatLog(fb, 500)

	// The hero killed at the FB moment: attacker 100.
	d2 := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
	d2.AttackerName = u32p(100)
	d2.IsTargetHero = boolp(true)
	a.handleCombatLog(d2, 500)

	bind(a, 0, 100) // slot 0 -> xpID 100 (FB hero)
	bind(a, 1, 200) // slot 1 -> xpID 200 (early death hero)

	pr := fakePR{slots: map[int]prSlot{
		0: {steamID: 100, team: teamRadiant, name: "FBHero", heroID: 1, heroHandle: 1},
		1: {steamID: 200, team: teamDire, name: "Early", heroID: 2, heroHandle: 2},
	}}
	heroes := map[uint64]heroInfo{
		1: {name: "FBHero", playerID: 0},
		2: {name: "Early", playerID: 2},
	}
	res := buildResult(a, pr, heroes, nil, 0)

	if len(res.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(res.Players))
	}
	if !res.Players[0].FirstBlood {
		t.Errorf("player 0 should claim first blood (type-18 attacker 100)")
	}
	if res.Players[1].FirstBlood {
		t.Errorf("player 1 (early DEATH attacker 200) must not claim first blood")
	}
}

// TestFirstBloodFallback checks the plain first hero DEATH is used when no
// FIRST_BLOOD (type 18) event is present in the replay.
func TestFirstBloodFallback(t *testing.T) {
	a := newAggregator()

	d1 := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
	d1.AttackerName = u32p(100)
	d1.IsTargetHero = boolp(true)
	a.handleCombatLog(d1, 500)

	d2 := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
	d2.AttackerName = u32p(101)
	d2.IsTargetHero = boolp(true)
	a.handleCombatLog(d2, 900)

	bind(a, 0, 100) // slot 0 -> xpID 100 (first hero DEATH attacker)
	bind(a, 1, 101) // slot 1 -> xpID 101

	pr := fakePR{slots: map[int]prSlot{
		0: {steamID: 100, team: teamRadiant, name: "A", heroID: 1, heroHandle: 1},
		1: {steamID: 200, team: teamDire, name: "B", heroID: 2, heroHandle: 2},
	}}
	heroes := map[uint64]heroInfo{
		1: {name: "A", playerID: 0},
		2: {name: "B", playerID: 2},
	}
	res := buildResult(a, pr, heroes, nil, 0)

	if !res.Players[0].FirstBlood {
		t.Errorf("player 0 (first hero DEATH attacker 100) should claim first blood")
	}
	if res.Players[1].FirstBlood {
		t.Errorf("player 1 must not claim first blood")
	}
}

// TestFirstBloodFromXpIDZero checks that a first-blood attacker with xpID 0
// (a valid PLAYERSTATS value, spec §6.3) still claims the credit. The old 0
// sentinel could neither record nor match such an attacker.
func TestFirstBloodFromXpIDZero(t *testing.T) {
	a := newAggregator()

	fb := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_FIRST_BLOOD)
	a.handleCombatLog(fb, 500)

	d := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
	d.AttackerName = u32p(0) // the FB attacker has xpID 0
	d.IsTargetHero = boolp(true)
	a.handleCombatLog(d, 500)

	bind(a, 0, 0) // slot 0 -> xpID 0 (the FB hero)
	bind(a, 1, 100)

	pr := fakePR{slots: map[int]prSlot{
		0: {steamID: 100, team: teamRadiant, name: "Zero", heroID: 1, heroHandle: 1},
		1: {steamID: 200, team: teamDire, name: "One", heroID: 2, heroHandle: 2},
	}}
	heroes := map[uint64]heroInfo{
		1: {name: "Zero", playerID: 0},
		2: {name: "One", playerID: 2},
	}
	res := buildResult(a, pr, heroes, nil, 0)

	if len(res.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(res.Players))
	}
	if !res.Players[0].FirstBlood {
		t.Errorf("player 0 (xpID 0 attacker) should claim first blood")
	}
	if res.Players[1].FirstBlood {
		t.Errorf("player 1 must not claim first blood")
	}
}

// TestNoFirstBloodEvents checks that a replay with no FIRST_BLOOD event and no
// hero DEATH claims no first blood for anyone (the presence flags stay false,
// so the 0 default fbXP cannot match the valid xpID 0 slot).
func TestNoFirstBloodEvents(t *testing.T) {
	a := newAggregator()
	bind(a, 0, 0)
	bind(a, 1, 100)

	pr := fakePR{slots: map[int]prSlot{
		0: {steamID: 100, team: teamRadiant, name: "A", heroID: 1, heroHandle: 1},
		1: {steamID: 200, team: teamDire, name: "B", heroID: 2, heroHandle: 2},
	}}
	res := buildResult(a, pr, nil, nil, 0)

	if len(res.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(res.Players))
	}
	for i := range res.Players {
		if res.Players[i].FirstBlood {
			t.Errorf("player %d must not claim first blood without events", i)
		}
	}
}

func TestTeamDataReader(t *testing.T) {
	slot := prSlot{steamID: 1, team: teamRadiant, teamSlot: 2, name: "x", heroID: 1, heroHandle: 1}
	heroes := map[uint64]heroInfo{1: {name: "x", playerID: 0}}
	teams := fakeTeam{td: map[string]map[int32]teamData{
		"radiant": {2: {campsStacked: 7, creepsStacked: 3}},
	}}
	a := newAggregator()
	p := buildPlayer(a, slot, heroes, teams, 0, 0)
	if p.CampsStacked != 7 {
		t.Errorf("camps_stacked = %d, want 7", p.CampsStacked)
	}
	if p.CreepsStacked != 3 {
		t.Errorf("creeps_stacked = %d, want 3", p.CreepsStacked)
	}
}

// TestScoreboardTeamData checks that when the scoreboard numbers are present
// (teamData.score), GPM/XPM, tower damage, healing and the consumable-gold
// counts come from them and not from the combat-log aggregates. Hero damage is
// the exception: it stays on the combat-log sum (the scoreboard m_flHeroDamage
// diverges from OpenDota), so a player with no combat-log events reports 0.
func TestScoreboardTeamData(t *testing.T) {
	slot := prSlot{
		steamID: 1, team: teamRadiant, teamSlot: 0, name: "x", heroID: 1, heroHandle: 1,
		kills: 2, deaths: 1, assists: 3, level: 20,
	}
	heroes := map[uint64]heroInfo{1: {name: "x", playerID: 0}}
	teams := fakeTeam{td: map[string]map[int32]teamData{
		"radiant": {0: {
			campsStacked:      7,
			creepsStacked:     3,
			totalEarnedGold:   23209,
			totalEarnedXP:     28298,
			towerDamage:       963,
			healing:           8804,
			netWorth:          21024,
			lastHitCount:      251,
			stuns:             12.5,
			runePickups:       1,
			sentryWardsPlaced: 12,
			smokesUsed:        4,
			score:             true,
		}},
	}}
	a := newAggregator()
	p := buildPlayer(a, slot, heroes, teams, 0, 2463)

	if p.GPM != 565 { // floor(23209*60/2463)
		t.Errorf("gpm = %d, want 565", p.GPM)
	}
	if p.XPM != 689 { // floor(28298*60/2463)
		t.Errorf("xpm = %d, want 689", p.XPM)
	}
	if p.TowerDamage != 963 {
		t.Errorf("tower_damage = %d, want 963", p.TowerDamage)
	}
	if p.Healing != 8804 {
		t.Errorf("healing = %v, want 8804", p.Healing)
	}
	if p.HeroDamage != 0 {
		t.Errorf("hero_damage = %d, want 0 (combat-log, no events)", p.HeroDamage)
	}
	if p.Networth != 21024 {
		t.Errorf("networth = %d, want 21024", p.Networth)
	}
	if p.LastHits != 251 {
		t.Errorf("last_hits = %d, want 251", p.LastHits)
	}
	if p.CampsStacked != 7 || p.CreepsStacked != 3 {
		t.Errorf("stacked = %d/%d, want 7/3", p.CampsStacked, p.CreepsStacked)
	}
	if p.RunePickups != 1 {
		t.Errorf("rune_pickups = %d, want 1", p.RunePickups)
	}
	if p.GoldWards != 600 { // 12 sentries * 50; observer wards are free
		t.Errorf("gold_spent_wards = %d, want 600", p.GoldWards)
	}
	if p.GoldSmoke != 200 { // 4 smokes * 50
		t.Errorf("gold_spent_smoke = %d, want 200", p.GoldSmoke)
	}
	if p.GoldDust != 0 {
		t.Errorf("gold_spent_dust = %d, want 0", p.GoldDust)
	}
}

// TestTeamDataReaderMissing ensures a missing team row leaves camps_stacked 0.
func TestTeamDataReaderMissing(t *testing.T) {
	slot := prSlot{steamID: 1, team: teamDire, teamSlot: 4, name: "x", heroID: 1, heroHandle: 1}
	heroes := map[uint64]heroInfo{1: {name: "x", playerID: 0}}
	teams := fakeTeam{td: map[string]map[int32]teamData{
		"radiant": {2: {campsStacked: 7}},
	}}
	a := newAggregator()
	p := buildPlayer(a, slot, heroes, teams, 0, 0)
	if p.CampsStacked != 0 {
		t.Errorf("camps_stacked = %d, want 0 for missing row", p.CampsStacked)
	}
}

// TestScoreboardSentinelFallback checks that a scoreboard field marked -1
// (absent on the patch) keeps the combat-log aggregate instead of overwriting
// it with a bogus zero: net worth and last hits stay on PLAYERSTATS, healing
// on the combat-log sum, GPM/XPM on the combat-log gold/xp. Fields that are
// present still win (tower damage, stuns, runes).
func TestScoreboardSentinelFallback(t *testing.T) {
	slot := prSlot{
		steamID: 1, team: teamRadiant, teamSlot: 0, name: "x", heroID: 1, heroHandle: 1,
	}
	heroes := map[uint64]heroInfo{1: {name: "x", playerID: 0}}
	teams := fakeTeam{td: map[string]map[int32]teamData{
		"radiant": {0: {
			campsStacked:    7,
			creepsStacked:   3,
			totalEarnedGold: 23209,
			towerDamage:     963,
			stuns:           12.5,
			runePickups:     1,
			// netWorth, lastHitCount, healing, totalEarnedXP stay -1 (absent).
			netWorth:      -1,
			lastHitCount:  -1,
			healing:       -1,
			totalEarnedXP: -1,
			score:         true,
		}},
	}}
	a := newAggregator()
	bind(a, 0, 100)

	// Combat-log aggregates: 230 last hits (PLAYERSTATS), 6000 gold, 200 heal.
	ps := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS)
	ps.TargetName = u32p(0)
	ps.Value = u32p(100)
	ps.LastHits = u32p(230)
	a.handleCombatLog(ps, 0)
	g := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD)
	g.TargetName = u32p(100)
	g.Value = u32p(6000)
	a.handleCombatLog(g, 0)
	h := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL)
	h.AttackerName = u32p(100)
	h.IsTargetHero = boolp(true)
	h.Value = u32p(200)
	a.handleCombatLog(h, 0)

	p := buildPlayer(a, slot, heroes, teams, 0, 2463)

	if p.LastHits != 230 {
		t.Errorf("last_hits = %d, want 230 (PLAYERSTATS fallback)", p.LastHits)
	}
	if p.Networth != 0 {
		t.Errorf("networth = %d, want 0 (absent scoreboard field)", p.Networth)
	}
	if p.Healing != 200 {
		t.Errorf("healing = %v, want 200 (combat-log fallback)", p.Healing)
	}
	// GPM comes from the present scoreboard totalEarnedGold (floor(23209*60/2463));
	// XPM falls back to the absent totalEarnedXP, i.e. the combat-log xp (0).
	if p.GPM != 565 { // floor(23209*60/2463)
		t.Errorf("gpm = %d, want 565 (present scoreboard field)", p.GPM)
	}
	if p.XPM != 0 {
		t.Errorf("xpm = %d, want 0 (no xp, absent scoreboard field)", p.XPM)
	}
	if p.TowerDamage != 963 {
		t.Errorf("tower_damage = %d, want 963 (present scoreboard field)", p.TowerDamage)
	}
	if p.StunDuration != 12.5 {
		t.Errorf("stun_duration = %v, want 12.5", p.StunDuration)
	}
	if p.RunePickups != 1 {
		t.Errorf("rune_pickups = %d, want 1", p.RunePickups)
	}
}

// TestHeroDamageAndDamageTaken checks combat-log hero damage: damage to a hero
// is credited to the attacker (self-damage excluded from hero_damage, matching
// OpenDota), damage taken counts attacks by heroes including self-inflicted
// damage (as OpenDota's hero-source sum does), and damage involving illusions
// is excluded from both sides.
func TestHeroDamageAndDamageTaken(t *testing.T) {
	a := newAggregator()

	// 100 (hero) deals 300 to 200 (hero).
	d := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
	d.AttackerName = u32p(100)
	d.TargetName = u32p(200)
	d.IsTargetHero = boolp(true)
	d.IsAttackerHero = boolp(true)
	d.Value = u32p(300)
	a.handleCombatLog(d, 0)

	// Self-damage: excluded from hero_damage, included in damage_taken.
	s := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
	s.AttackerName = u32p(100)
	s.TargetName = u32p(100)
	s.IsTargetHero = boolp(true)
	s.IsAttackerHero = boolp(true)
	s.Value = u32p(50)
	a.handleCombatLog(s, 0)

	// A creep (900, not a hero) damages 200 for 40: it is not hero-vs-hero,
	// so 200's damage_taken must stay 300.
	c := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
	c.AttackerName = u32p(900)
	c.TargetName = u32p(200)
	c.IsTargetHero = boolp(true)
	c.Value = u32p(40)
	a.handleCombatLog(c, 0)

	// Damage to an illusion of 200 (target is a hero unit but illusion-flagged)
	// counts for neither hero_damage nor damage_taken.
	i := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE)
	i.AttackerName = u32p(100)
	i.TargetName = u32p(200)
	i.IsTargetHero = boolp(true)
	i.IsAttackerHero = boolp(true)
	i.IsTargetIllusion = boolp(true)
	i.Value = u32p(999)
	a.handleCombatLog(i, 0)

	if x := a.xpStats(100).heroDamage; x != 300 {
		t.Errorf("hero_damage[100] = %d, want 300", x)
	}
	if x := a.xpStats(200).damageTaken; x != 300 {
		t.Errorf("damage_taken[200] = %d, want 300", x)
	}
	if x := a.xpStats(100).damageTaken; x != 50 {
		t.Errorf("damage_taken[100] = %d, want 50 (self-damage counted)", x)
	}
}

// TestPurchaseGold checks ITEM events resolve item names against the injected
// resolver and bucket the gold into wards/smoke/dust using the constant prices
// (ITEM events carry no gold value on modern patches). Observer wards are free
// and the gem is absent from the ward bucket (its id never appears in the
// combat log on modern patches), so both count as 0 gold.
func TestPurchaseGold(t *testing.T) {
	a := newAggregator()
	names := map[int32]string{
		1: "item_ward_observer",
		2: "item_ward_sentry",
		3: "item_gem",
		4: "item_smoke_of_deceit",
		5: "item_dust",
		6: "item_quelling_blade",
	}
	a.lookup = func(table string, index int32) (string, bool) {
		if table != "CombatLogNames" {
			return "", false
		}
		n, ok := names[index]
		return n, ok
	}

	buy := func(item int32) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_ITEM)
		e.AttackerName = u32p(100)
		e.InflictorName = u32p(uint32(item))
		a.handleCombatLog(e, 0)
	}
	buy(1) // obs ward (free, no gold)
	buy(2) // sentry
	buy(3) // gem
	buy(4) // smoke
	buy(5) // dust
	buy(5) // dust
	buy(6) // not a tracked consumable

	x := a.xpStats(100)
	if x.goldWards != 50 { // 1 sentry * 50; obs and gem count as 0
		t.Errorf("goldWards = %d, want 50", x.goldWards)
	}
	if x.goldSmoke != 50 {
		t.Errorf("goldSmoke = %d, want 50", x.goldSmoke)
	}
	if x.goldDust != 160 {
		t.Errorf("goldDust = %d, want 160", x.goldDust)
	}
}

// TestDeadTimeTracking checks m_flRespawnTime transitions paired with the
// combat-log DEATH clocks produce the spent-dead time: the buyback cap,
// heroes still dead at the end of the match, multiple buybacks per slot (each
// capping its own window) and the missing post-game clock falling back to the
// respawn cap instead of zeroing the window.
func TestDeadTimeTracking(t *testing.T) {
	gameState := func(a *aggregator, value, ts float32) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE)
		e.Value = u32p(uint32(value))
		e.Timestamp = f32p(ts)
		a.handleCombatLog(e, 0)
	}
	death := func(a *aggregator, xpID, ts float32) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH)
		e.TargetName = u32p(uint32(xpID))
		e.Timestamp = f32p(ts)
		e.IsTargetHero = boolp(true)
		a.handleCombatLog(e, 0)
	}
	buyback := func(a *aggregator, slot, ts float32) {
		e := entry(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_BUYBACK)
		e.Value = u32p(uint32(slot))
		e.Timestamp = f32p(ts)
		a.handleCombatLog(e, 0)
	}

	t.Run("single buyback and post-game cap", func(t *testing.T) {
		a := newAggregator()
		gameState(a, 5, 100.0)
		gameState(a, 6, 700.0) // post-game clock 700.0

		// Hero 7 dies at 123.5; initial m_flRespawnTime is -1 (alive), death
		// sets it to the scheduled respawn 200.0.
		a.respawnUpdate(7, -1)
		a.respawnUpdate(7, 200.0)
		// xpID 100 is the player's combat-log id; a buyback at 150.0 caps the
		// window to 26.5 s.
		death(a, 100, 123.5)
		buyback(a, 3, 150.0)

		// Dies again at 400.0 with scheduled respawn 720.0, past the post-game
		// clock 700.0: counted only until 700.0 (300 s).
		a.respawnUpdate(7, 720.0)
		death(a, 100, 400.0)

		if got := a.timeDead(7, 100, 3); got > 326.501 || got < 326.499 {
			t.Errorf("timeDead = %v, want ~326.5", got)
		}
	})

	t.Run("multiple buybacks cap successive windows", func(t *testing.T) {
		a := newAggregator()
		gameState(a, 5, 100.0)
		gameState(a, 6, 1000.0)

		// Two deaths, each followed by its own buyback. The buggy reader kept
		// only the last buyback (350.0), which would also cap window 1 at
		// 350.0 (not < 200.0, so no cap) and overstate it to 100 s.
		a.respawnUpdate(7, -1)
		a.respawnUpdate(7, 200.0)
		death(a, 100, 100.0)
		buyback(a, 3, 150.0)

		a.respawnUpdate(7, 400.0)
		death(a, 100, 300.0)
		buyback(a, 3, 350.0)

		if got := a.timeDead(7, 100, 3); got > 100.001 || got < 99.999 {
			t.Errorf("timeDead = %v, want ~100", got)
		}
	})

	t.Run("missing post-game clock falls back to respawn", func(t *testing.T) {
		a := newAggregator()
		gameState(a, 5, 100.0)
		// No GAME_STATE 6 event: a.postGame stays 0. The buggy reader clamped
		// every window to 0 and reported 0 dead time.

		a.respawnUpdate(7, -1)
		a.respawnUpdate(7, 900.0)
		death(a, 100, 500.0)
		buyback(a, 3, 600.0)

		if got := a.timeDead(7, 100, 3); got > 100.001 || got < 99.999 {
			t.Errorf("timeDead = %v, want ~100 (respawn/buyback cap)", got)
		}
	})
}

func TestMatchIDFromFilename(t *testing.T) {
	cases := map[string]uint64{
		"7047839963_12345.dem.bz2": 7047839963,
		"8552595443.dem":           8552595443,
		"/tmp/replays/42_foo.dem":  42,
		"replay.dem":               0,
		"/tmp/replays/foo.dem.bz2": 0,
		"0123_bar.dem":             123,
	}
	for in, want := range cases {
		if got := matchIDFromFilename(in); got != want {
			t.Errorf("matchIDFromFilename(%q) = %d, want %d", in, got, want)
		}
	}
}
