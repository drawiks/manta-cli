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
