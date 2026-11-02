package stats

import "testing"

func TestClassifyModifier(t *testing.T) {
	cases := []struct {
		inf      string
		kind     modKind
		category string
		clean    string
	}{
		{"modifier_dazzle_shallow_grave", kindBuff, "save", "dazzle_shallow_grave"},
		{"modifier_axe_berserkers_call", kindTaunt, "", "axe_berserkers_call"},
		{"modifier_heavens_halberd", kindDisarm, "", "heavens_halberd"},
		{"modifier_item_urn_heal", kindHeal, "", "item_urn_heal"},
		{"modifier_silver_edge_debuff", kindBreak, "", "silver_edge"},
		{"modifier_naga_siren_ensnare", kindRoot, "", "naga_siren_ensnare"},
	}
	for _, c := range cases {
		got := classifyModifier(c.inf)
		if got.kind != c.kind || got.category != c.category || got.clean != c.clean {
			t.Errorf("%s: got kind=%v cat=%q clean=%q, want kind=%v cat=%q clean=%q",
				c.inf, got.kind, got.category, got.clean, c.kind, c.category, c.clean)
		}
	}
}

func TestAggregateCombatAndIntervals(t *testing.T) {
	a := newAggregator()
	a.setEpilogue(42, 100, true, []playerInfo{
		{Slot: 0, SteamID: 1, Hero: "npc_dota_hero_dazzle", Name: "d", Team: "radiant"},
		{Slot: 1, SteamID: 2, Hero: "npc_dota_hero_lich", Name: "l", Team: "radiant"},
		{Slot: 2, SteamID: 3, Hero: "npc_dota_hero_axe", Name: "a", Team: "dire"},
	})
	for slot, team := range []string{"radiant", "radiant", "dire"} {
		a.setSlot(slot, team, a.roster[slot].Hero)
	}

	for _, iv := range []intervalData{
		{Slot: 0, Time: 0, HasLifeState: true, LifeState: 0},
		{Slot: 2, Time: 0, HasLifeState: true, LifeState: 0},
		{Slot: 0, Time: 50, HasLifeState: true, LifeState: 0},
		{Slot: 2, Time: 50, HasLifeState: true, LifeState: 2},
		{Slot: 0, Time: 100, HasLifeState: true, LifeState: 0, Gold: 1000, XP: 2000, LastHits: 10, Networth: 5000, Stuns: 12.5, Kills: 3, Deaths: 1, Assists: 2, Level: 10, CampsStacked: 1, CreepsStacked: 2, RunePickups: 3, WisdomsCaptured: 1, WatchersCaptured: 2, LotusesGathered: 3, SentryWards: 2},
		{Slot: 2, Time: 100, HasLifeState: true, LifeState: 0, Gold: 2000, XP: 3000, LastHits: 20, Networth: 9000, Stuns: 4, Level: 11},
	} {
		a.onInterval(iv)
	}

	// Ally buff: Dazzle shallow-grave on Lich for 5 seconds.
	a.onModifier(combatEvent{Time: 10, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "modifier_dazzle_shallow_grave", AttackerHero: true, TargetHero: true}, true)
	a.onModifier(combatEvent{Time: 15, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "modifier_dazzle_shallow_grave", AttackerHero: true, TargetHero: true}, false)

	// Enemy root: Axe rooted for 4 seconds.
	a.onModifier(combatEvent{Time: 20, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true}, true)
	a.onModifier(combatEvent{Time: 24, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true}, false)

	// Ally heal plus an instant item heal.
	a.onHeal(combatEvent{Time: 30, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "dota_unknown", AttackerHero: true, TargetHero: true, Value: 100})
	a.onHeal(combatEvent{Time: 31, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "item_famango", AttackerHero: true, TargetHero: true, Value: 50})

	a.onGold(combatEvent{Time: 40, TargetName: "npc_dota_hero_axe", Value: -200, GoldReason: 1})
	a.onPurchase(combatEvent{Time: 41, TargetName: "npc_dota_hero_dazzle", ValueName: "item_smoke_of_deceit"})
	a.onPurchase(combatEvent{Time: 42, TargetName: "npc_dota_hero_dazzle", ValueName: "item_dust"})
	a.onDeath(combatEvent{Time: 43, TargetName: "npc_dota_courier", SourceName: "npc_dota_hero_dazzle"})
	a.onFirstBlood(2)

	res := a.result()
	if res.MatchID != 42 || res.DurationSec != 100 || !res.RadiantWin {
		t.Fatalf("match header = %+v", res)
	}
	if len(res.Players) != 3 {
		t.Fatalf("players = %d, want 3", len(res.Players))
	}
	dazzle, lich, axe := res.Players[0], res.Players[1], res.Players[2]
	if dazzle.Hero != "dazzle" || dazzle.HeroID != 50 || dazzle.Team != "radiant" {
		t.Errorf("dazzle identity = %+v", dazzle)
	}
	if dazzle.GPM != 600 || dazzle.XPM != 1200 || dazzle.StunDuration != 12.5 {
		t.Errorf("dazzle interval stats = gpm %d xpm %d stun %v", dazzle.GPM, dazzle.XPM, dazzle.StunDuration)
	}
	if dazzle.BuffDuration != 5 || len(dazzle.BuffSources) != 1 || dazzle.BuffSources[0].Duration != 5 {
		t.Errorf("dazzle buffs = %+v", dazzle.BuffSources)
	}
	if dazzle.Healing != 150 || dazzle.HealValue != 50 || len(dazzle.HealSources) != 1 {
		t.Errorf("dazzle heals = healing %d value %v sources %+v", dazzle.Healing, dazzle.HealValue, dazzle.HealSources)
	}
	if dazzle.GoldSpentSmoke != 50 || dazzle.GoldSpentDust != 80 || dazzle.CourierKills != 1 {
		t.Errorf("dazzle purchases = smoke %d dust %d couriers %d", dazzle.GoldSpentSmoke, dazzle.GoldSpentDust, dazzle.CourierKills)
	}
	if dazzle.Position != 0 || dazzle.Lane != "" {
		t.Errorf("dazzle without samples should have no role, got %d %q", dazzle.Position, dazzle.Lane)
	}
	if axe.GoldLost != 200 || axe.TimeDead != 50 || !axe.FirstBlood {
		t.Errorf("axe = gold_lost %d time_dead %v first_blood %v", axe.GoldLost, axe.TimeDead, axe.FirstBlood)
	}
	if dazzle.RootsDuration != 4 || len(dazzle.RootSources) != 1 || dazzle.RootSources[0].Inflictor != "naga_siren_ensnare" {
		t.Errorf("dazzle roots = %v %+v", dazzle.RootsDuration, dazzle.RootSources)
	}
	if lich.Healing != 0 || lich.BuffDuration != 0 {
		t.Errorf("lich = healing %d buffs %v", lich.Healing, lich.BuffDuration)
	}
}
