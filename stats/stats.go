package stats

// Match is the parsed statistics for one Dota 2 match.
//
// Its JSON encoding is the odota-cli match object plus the mana_restored
// extension and the client-sourced gold_lost. The zero value is a match with
// no players and zero values.
type Match struct {
	// MatchID is the Dota 2 match ID, or zero when the replay has none.
	MatchID int64 `json:"match_id"`
	// DurationSec is the match duration in seconds.
	DurationSec float64 `json:"duration_sec"`
	// RadiantWin reports whether the Radiant team won.
	RadiantWin bool `json:"radiant_win"`
	// Players contains one entry per player, ordered by PlayerID.
	Players []Player `json:"players"`
}

// Player is the statistics for one player in a match.
//
// Durations are in seconds and gold values are in gold. The zero value is a
// player with unknown identity and zero statistics.
type Player struct {
	// SteamID is the player's 64-bit Steam account ID.
	SteamID int64 `json:"steam_id"`
	// PlayerID is the zero-based player slot, from 0 to 9.
	PlayerID int `json:"player_id"`
	// HeroID is the numeric Dota 2 hero ID.
	HeroID int `json:"hero_id"`
	// Hero is the hero slug without the npc_dota_hero_ prefix.
	Hero string `json:"hero"`
	// Team is "radiant" or "dire".
	Team string `json:"team"`
	// Name is the player name stored in the replay.
	Name string `json:"name"`
	// Position is the role from 1 to 5, or zero when there are not enough
	// position samples. It is omitted when zero.
	Position int `json:"position,omitempty"`
	// Lane is "safe", "mid", "off", "supp" or "hard_supp", or empty when
	// unknown. It is omitted when empty.
	Lane string `json:"lane,omitempty"`
	// Kills is the number of kills.
	Kills int `json:"kills"`
	// Deaths is the number of deaths.
	Deaths int `json:"deaths"`
	// Assists is the number of assists.
	Assists int `json:"assists"`
	// Level is the final hero level.
	Level int `json:"level"`
	// LastHits is the final last-hit count.
	LastHits int `json:"last_hits"`
	// Networth is the final net worth in gold.
	Networth int `json:"networth"`
	// GPM is gold per minute.
	GPM int `json:"gpm"`
	// XPM is experience per minute.
	XPM int `json:"xpm"`
	// Healing is the healing done to allies.
	Healing int `json:"healing"`
	// ManaRestored is the mana restored to allied heroes, including Bottle healing over time.
	ManaRestored int `json:"mana_restored"`
	// HeroDamage is the damage dealt to heroes.
	HeroDamage int `json:"hero_damage"`
	// DamageTaken is the damage taken from heroes.
	DamageTaken int `json:"damage_taken"`
	// TowerDamage is the damage dealt to buildings.
	TowerDamage int `json:"tower_damage"`
	// TimeDead is the total time spent dead in seconds.
	TimeDead float64 `json:"time_dead"`
	// StunDuration is the total stun duration in seconds.
	StunDuration float64 `json:"stun_duration"`
	// StunSources breaks StunDuration down by inflictor. It is omitted when
	// empty.
	StunSources []SourceEntry `json:"stun_sources,omitempty"`
	// BuffDuration is the total duration of tracked buffs in seconds.
	BuffDuration float64 `json:"buff_duration"`
	// BuffSources breaks BuffDuration down by inflictor and category. It is
	// omitted when empty.
	BuffSources []SourceEntry `json:"buff_sources,omitempty"`
	// FearDuration is the total fear duration in seconds.
	FearDuration float64 `json:"fear_duration"`
	// FearSources breaks FearDuration down by inflictor. It is omitted when
	// empty.
	FearSources []SourceEntry `json:"fear_sources,omitempty"`
	// RootsDuration is the total root duration in seconds.
	RootsDuration float64 `json:"roots_duration"`
	// RootSources breaks RootsDuration down by inflictor. It is omitted when
	// empty.
	RootSources []SourceEntry `json:"root_sources,omitempty"`
	// LeashDuration is the total leash duration in seconds.
	LeashDuration float64 `json:"leash_duration"`
	// LeashSources breaks LeashDuration down by inflictor. It is omitted when
	// empty.
	LeashSources []SourceEntry `json:"leash_sources,omitempty"`
	// TrapDuration is the total trap duration in seconds.
	TrapDuration float64 `json:"trap_duration"`
	// TrapSources breaks TrapDuration down by inflictor. It is omitted when
	// empty.
	TrapSources []SourceEntry `json:"trap_sources,omitempty"`
	// TauntDuration is the total taunt duration in seconds.
	TauntDuration float64 `json:"taunt_duration"`
	// TauntSources breaks TauntDuration down by inflictor. It is omitted when
	// empty.
	TauntSources []SourceEntry `json:"taunt_sources,omitempty"`
	// SilenceDuration is the total silence duration in seconds.
	SilenceDuration float64 `json:"silence_duration"`
	// SilenceSources breaks SilenceDuration down by inflictor. It is omitted
	// when empty.
	SilenceSources []SourceEntry `json:"silence_sources,omitempty"`
	// BreakDuration is the total break duration in seconds.
	BreakDuration float64 `json:"break_duration"`
	// BreakSources breaks BreakDuration down by inflictor. It is omitted when
	// empty.
	BreakSources []SourceEntry `json:"break_sources,omitempty"`
	// DisarmDuration is the total disarm duration in seconds.
	DisarmDuration float64 `json:"disarm_duration"`
	// DisarmSources breaks DisarmDuration down by inflictor. It is omitted
	// when empty.
	DisarmSources []SourceEntry `json:"disarm_sources,omitempty"`
	// HealDuration is the total duration of heal-over-time effects in seconds.
	HealDuration float64 `json:"heal_duration"`
	// HealValue is the healing amount from heal-over-time and instant heals.
	HealValue float64 `json:"heal_value"`
	// HealSources breaks healing down by inflictor. It is omitted when empty.
	HealSources []SourceEntry `json:"heal_sources,omitempty"`
	// GoldSpentWards is the gold spent on wards.
	GoldSpentWards int `json:"gold_spent_wards"`
	// GoldSpentSmoke is the gold spent on Smoke of Deceit.
	GoldSpentSmoke int `json:"gold_spent_smoke"`
	// GoldSpentDust is the gold spent on Dust of Appearance.
	GoldSpentDust int `json:"gold_spent_dust"`
	// GoldLost is the total gold lost to deaths reported by the game client.
	GoldLost int `json:"gold_lost"`
	// CampsStacked is the number of stacked neutral camps.
	CampsStacked int `json:"camps_stacked"`
	// CreepsStacked is the number of stacked creep camps.
	CreepsStacked int `json:"creeps_stacked"`
	// RunePickups is the number of runes picked up.
	RunePickups int `json:"rune_pickups"`
	// WisdomsCaptured is the number of Wisdom runes captured.
	WisdomsCaptured int `json:"wisdoms_captured"`
	// WatchersCaptured is the number of Watchers captured.
	WatchersCaptured int `json:"watchers_captured"`
	// LotusesGathered is the number of Lotuses gathered.
	LotusesGathered int `json:"lotuses_gathered"`
	// FirstBlood reports whether the player drew first blood.
	FirstBlood bool `json:"first_blood"`
	// CourierKills is the number of enemy couriers killed.
	CourierKills int `json:"courier_kills"`
}

// SourceEntry is one inflictor's contribution to a tracked effect.
//
// It records either a duration, a value or a count, depending on the effect.
// The zero value is an empty entry.
type SourceEntry struct {
	// Inflictor is the normalized ability, item or modifier name.
	Inflictor string `json:"inflictor"`
	// Category is the effect category, such as "stun", "purge", "heal",
	// "fear", "root", "leash", "trap", "taunt", "silence", "break" or
	// "disarm".
	Category string `json:"category"`
	// Duration is the effect duration in seconds. It is omitted when zero.
	Duration float64 `json:"duration,omitempty"`
	// Value is the healing value or other effect amount. It is omitted when
	// zero.
	Value float64 `json:"value,omitempty"`
	// Count is the number of applications. It is omitted when zero.
	Count int `json:"count,omitempty"`
}
