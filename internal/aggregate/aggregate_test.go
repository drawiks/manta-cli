package aggregate

import (
	"testing"

	"github.com/drawiks/manta-cli/internal/events"
)

func TestAggregateCombatAndIntervals(t *testing.T) {
	a := New()
	a.Epilogue(events.Epilogue{
		MatchID:     42,
		DurationSec: 100,
		RadiantWin:  true,
		Players: []events.PlayerInfo{
			{Slot: 0, SteamID: 1, Hero: "npc_dota_hero_dazzle", Name: "d", Team: "radiant"},
			{Slot: 1, SteamID: 2, Hero: "npc_dota_hero_lich", Name: "l", Team: "radiant"},
			{Slot: 2, SteamID: 3, Hero: "npc_dota_hero_axe", Name: "a", Team: "dire"},
		},
	})
	for slot, team := range []string{"radiant", "radiant", "dire"} {
		a.Slot(events.Slot{Slot: slot, Team: team, Hero: a.roster[slot].Hero})
	}

	for _, iv := range []events.Interval{
		{Slot: 0, Time: 0, HasLifeState: true, LifeState: 0},
		{Slot: 2, Time: 0, HasLifeState: true, LifeState: 0},
		{Slot: 0, Time: 50, HasLifeState: true, LifeState: 0},
		{Slot: 2, Time: 50, HasLifeState: true, LifeState: 2},
		{Slot: 0, Time: 100, HasLifeState: true, LifeState: 0, Gold: 1000, XP: 2000, LastHits: 10, Networth: 5000, Stuns: 12.5, Kills: 3, Deaths: 1, Assists: 2, Level: 10, CampsStacked: 1, CreepsStacked: 2, RunePickups: 3, WisdomsCaptured: 1, WatchersCaptured: 2, LotusesGathered: 3, SentryWards: 2},
		{Slot: 2, Time: 100, HasLifeState: true, LifeState: 0, Gold: 2000, XP: 3000, LastHits: 20, Networth: 9000, Stuns: 4, Level: 11},
	} {
		a.Interval(iv)
	}

	combat := func(ev events.Combat) {
		t.Helper()
		a.Combat(ev)
	}
	combat(events.Combat{Time: 10, Type: events.CombatModifierAdd, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "modifier_dazzle_shallow_grave", AttackerHero: true, TargetHero: true})
	combat(events.Combat{Time: 15, Type: events.CombatModifierRemove, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "modifier_dazzle_shallow_grave", AttackerHero: true, TargetHero: true})
	combat(events.Combat{Time: 20, Type: events.CombatModifierAdd, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true})
	combat(events.Combat{Time: 24, Type: events.CombatModifierRemove, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true})
	combat(events.Combat{Time: 30, Type: events.CombatHeal, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "dota_unknown", AttackerHero: true, TargetHero: true, Value: 100})
	combat(events.Combat{Time: 31, Type: events.CombatHeal, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_lich", Inflictor: "item_famango", AttackerHero: true, TargetHero: true, Value: 50})
	combat(events.Combat{Time: 40, Type: events.CombatGold, TargetName: "npc_dota_hero_axe", Value: -200, GoldReason: 1})
	combat(events.Combat{Time: 41, Type: events.CombatPurchase, TargetName: "npc_dota_hero_dazzle", ValueName: "item_smoke_of_deceit"})
	combat(events.Combat{Time: 42, Type: events.CombatPurchase, TargetName: "npc_dota_hero_dazzle", ValueName: "item_dust"})
	combat(events.Combat{Time: 43, Type: events.CombatDeath, TargetName: "npc_dota_courier", SourceName: "npc_dota_hero_dazzle"})
	a.Chat(events.Chat{Kind: events.ChatFirstBlood, Slot: 2})

	res := a.Result()
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

func TestSaveAndPurgeClassification(t *testing.T) {
	a := New()
	a.Epilogue(events.Epilogue{Players: []events.PlayerInfo{
		{Slot: 0, Hero: "npc_dota_hero_oracle", Team: "dire"},
		{Slot: 1, Hero: "npc_dota_hero_lion", Team: "dire"},
		{Slot: 2, Hero: "npc_dota_hero_axe", Team: "radiant"},
	}})
	for slot, team := range []string{"dire", "dire", "radiant"} {
		a.Slot(events.Slot{Slot: slot, Team: team, Hero: a.roster[slot].Hero})
	}

	modifier := func(add bool, at int, inflictor string) {
		t.Helper()
		kind := events.CombatModifierRemove
		if add {
			kind = events.CombatModifierAdd
		}
		a.Combat(events.Combat{
			Time:         at,
			Type:         kind,
			AttackerName: "npc_dota_hero_oracle",
			TargetName:   "npc_dota_hero_lion",
			Inflictor:    inflictor,
			AttackerHero: true,
			TargetHero:   true,
		})
	}

	modifier(true, 100, "modifier_oracle_false_promise_timer")
	modifier(false, 107, "modifier_oracle_false_promise_timer")
	modifier(true, 110, "modifier_wind_waker")
	modifier(false, 115, "modifier_wind_waker")
	modifier(true, 120, "modifier_disperser_movespeed_buff")
	modifier(true, 130, "modifier_eul_cyclone")
	modifier(false, 135, "modifier_eul_cyclone")
	modifier(true, 140, "modifier_pugna_decrepify")
	modifier(false, 145, "modifier_pugna_decrepify")
	modifier(true, 150, "modifier_underlord_dark_rift")
	modifier(false, 155, "modifier_underlord_dark_rift")
	modifier(true, 160, "modifier_abaddon_borrowed_time")
	modifier(false, 165, "modifier_abaddon_borrowed_time")

	res := a.Result()
	oracle := res.Players[0]
	if oracle.BuffDuration != 12 {
		t.Errorf("oracle buff_duration = %v, want 12", oracle.BuffDuration)
	}

	saves := map[string]float64{}
	purges := map[string]int{}
	for _, s := range oracle.BuffSources {
		switch s.Category {
		case "save":
			saves[s.Inflictor] = s.Duration
		case "purge":
			purges[s.Inflictor] += s.Count
		}
	}
	if len(saves) != 2 || saves["oracle_false_promise"] != 7 || saves["wind_waker"] != 5 {
		t.Errorf("oracle save sources = %+v", oracle.BuffSources)
	}
	if purges["disperser_movespeed"] != 1 {
		t.Errorf("oracle purge sources = %+v", oracle.BuffSources)
	}
}

func TestUndyingTombstoneBunkerSave(t *testing.T) {
	a := New()
	a.Epilogue(events.Epilogue{Players: []events.PlayerInfo{
		{Slot: 0, Hero: "npc_dota_hero_undying", Team: "dire"},
		{Slot: 1, Hero: "npc_dota_hero_pudge", Team: "dire"},
		{Slot: 2, Hero: "npc_dota_hero_axe", Team: "radiant"},
	}})
	for slot, team := range []string{"dire", "dire", "radiant"} {
		a.Slot(events.Slot{Slot: slot, Team: team, Hero: a.roster[slot].Hero})
	}

	modifier := func(add bool, at int, target, inflictor string) {
		t.Helper()
		kind := events.CombatModifierRemove
		if add {
			kind = events.CombatModifierAdd
		}
		a.Combat(events.Combat{
			Time:         at,
			Type:         kind,
			AttackerName: "npc_dota_hero_undying",
			TargetName:   target,
			Inflictor:    inflictor,
			AttackerHero: true,
			TargetHero:   true,
		})
	}

	modifier(true, 100, "npc_dota_hero_pudge", "modifier_undying_tombstone_bunker_friendly")
	modifier(false, 105, "npc_dota_hero_pudge", "modifier_undying_tombstone_bunker_friendly")
	modifier(true, 100, "npc_dota_hero_pudge", "modifier_undying_tombstone_bunker_cooldown")
	modifier(false, 102, "npc_dota_hero_pudge", "modifier_undying_tombstone_bunker_cooldown")
	modifier(true, 110, "npc_dota_hero_undying", "modifier_undying_tombstone_bunker_friendly")
	modifier(false, 115, "npc_dota_hero_undying", "modifier_undying_tombstone_bunker_friendly")

	res := a.Result()
	undying := res.Players[0]
	if undying.BuffDuration != 5 {
		t.Errorf("undying buff_duration = %v, want 5", undying.BuffDuration)
	}
	if len(undying.BuffSources) != 1 {
		t.Fatalf("undying buff_sources = %+v", undying.BuffSources)
	}
	source := undying.BuffSources[0]
	if source.Inflictor != "undying_tombstone_bunker_friendly" || source.Category != "save" || source.Duration != 5 {
		t.Errorf("undying buff source = %+v", source)
	}
}

func BenchmarkAggregate(b *testing.B) {
	a := New()
	a.Epilogue(events.Epilogue{
		MatchID:     1,
		DurationSec: 2400,
		Players: []events.PlayerInfo{
			{Slot: 0, Hero: "npc_dota_hero_dazzle", Team: "radiant"},
			{Slot: 1, Hero: "npc_dota_hero_axe", Team: "dire"},
		},
	})
	a.Slot(events.Slot{Slot: 0, Team: "radiant", Hero: "npc_dota_hero_dazzle"})
	a.Slot(events.Slot{Slot: 1, Team: "dire", Hero: "npc_dota_hero_axe"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Interval(events.Interval{Slot: 0, Time: i, HasLifeState: true, Gold: i, XP: i, Networth: i})
		a.Combat(events.Combat{Time: i, Type: events.CombatModifierAdd, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true})
		a.Combat(events.Combat{Time: i + 1, Type: events.CombatModifierRemove, AttackerName: "npc_dota_hero_dazzle", TargetName: "npc_dota_hero_axe", Inflictor: "modifier_naga_siren_ensnare", AttackerHero: true, TargetHero: true})
	}
}
