package stats

// Result is the top-level JSON object emitted to stdout.
type Result struct {
	MatchID     uint64   `json:"match_id,omitempty"`
	DurationSec int32    `json:"duration_sec"`
	RadiantWin  bool     `json:"radiant_win"`
	Players     []Player `json:"players"`
}

// Player holds aggregated per-player statistics.
type Player struct {
	SteamID       uint64       `json:"steam_id"`
	PlayerID      int32        `json:"player_id"`
	HeroID        int32        `json:"hero_id"`
	Hero          string       `json:"hero"`
	Team          string       `json:"team"`
	Name          string       `json:"name"`
	Kills         int32        `json:"kills"`
	Deaths        int32        `json:"deaths"`
	Assists       int32        `json:"assists"`
	Level         int32        `json:"level"`
	LastHits      uint32       `json:"last_hits"`
	Networth      uint32       `json:"networth"`
	GPM           uint32       `json:"gpm"`
	XPM           uint32       `json:"xpm"`
	Healing       float64      `json:"healing"`
	HeroDamage    int64        `json:"hero_damage"`
	DamageTaken   int64        `json:"damage_taken"`
	TowerDamage   int64        `json:"tower_damage"`
	TimeDead      float64      `json:"time_dead"`
	StunDuration  float64      `json:"stun_duration"`
	StunSources   []StunSource `json:"stun_sources,omitempty"`
	BuffSources   []BuffSource `json:"buff_sources,omitempty"`
	BuffsDuration float64      `json:"buffs_duration"`
	GoldWards     int64        `json:"gold_spent_wards"`
	GoldSmoke     int64        `json:"gold_spent_smoke"`
	GoldDust      int64        `json:"gold_spent_dust"`
	CampsStacked  int64        `json:"camps_stacked"`
	CreepsStacked int64        `json:"creeps_stacked"`
	RunePickups   int32        `json:"rune_pickups"`
	FirstBlood    bool         `json:"first_blood"`
}

// StunSource is one line of stun_duration attribution (modifier / scoreboard).
type StunSource struct {
	Inflictor string  `json:"inflictor"`
	Category  string  `json:"category"`
	Seconds   float64 `json:"seconds"`
}

// BuffSource is one line of positive-effect attribution (buffs a player applied,
// on allies or on itself). Value is uptime seconds for duration buffs, healed
// HP for heal, and the application/event count is in Count.
type BuffSource struct {
	Inflictor string  `json:"inflictor"`
	Category  string  `json:"category"`
	Value     float64 `json:"value"`
	Count     int     `json:"count"`
}
