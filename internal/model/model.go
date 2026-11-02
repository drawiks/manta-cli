package model

type Result struct {
	MatchID     int64    `json:"match_id"`
	DurationSec float64  `json:"duration_sec"`
	RadiantWin  bool     `json:"radiant_win"`
	Players     []Player `json:"players"`
}

type Player struct {
	SteamID          int64         `json:"steam_id"`
	PlayerID         int           `json:"player_id"`
	HeroID           int           `json:"hero_id"`
	Hero             string        `json:"hero"`
	Team             string        `json:"team"`
	Name             string        `json:"name"`
	Position         int           `json:"position,omitempty"`
	Lane             string        `json:"lane,omitempty"`
	Kills            int           `json:"kills"`
	Deaths           int           `json:"deaths"`
	Assists          int           `json:"assists"`
	Level            int           `json:"level"`
	LastHits         int           `json:"last_hits"`
	Networth         int           `json:"networth"`
	GPM              int           `json:"gpm"`
	XPM              int           `json:"xpm"`
	Healing          int           `json:"healing"`
	HeroDamage       int           `json:"hero_damage"`
	DamageTaken      int           `json:"damage_taken"`
	TowerDamage      int           `json:"tower_damage"`
	TimeDead         float64       `json:"time_dead"`
	StunDuration     float64       `json:"stun_duration"`
	StunSources      []SourceEntry `json:"stun_sources,omitempty"`
	BuffDuration     float64       `json:"buff_duration"`
	BuffSources      []SourceEntry `json:"buff_sources,omitempty"`
	FearDuration     float64       `json:"fear_duration"`
	FearSources      []SourceEntry `json:"fear_sources,omitempty"`
	RootsDuration    float64       `json:"roots_duration"`
	RootSources      []SourceEntry `json:"root_sources,omitempty"`
	LeashDuration    float64       `json:"leash_duration"`
	LeashSources     []SourceEntry `json:"leash_sources,omitempty"`
	TrapDuration     float64       `json:"trap_duration"`
	TrapSources      []SourceEntry `json:"trap_sources,omitempty"`
	TauntDuration    float64       `json:"taunt_duration"`
	TauntSources     []SourceEntry `json:"taunt_sources,omitempty"`
	SilenceDuration  float64       `json:"silence_duration"`
	SilenceSources   []SourceEntry `json:"silence_sources,omitempty"`
	BreakDuration    float64       `json:"break_duration"`
	BreakSources     []SourceEntry `json:"break_sources,omitempty"`
	DisarmDuration   float64       `json:"disarm_duration"`
	DisarmSources    []SourceEntry `json:"disarm_sources,omitempty"`
	HealDuration     float64       `json:"heal_duration"`
	HealValue        float64       `json:"heal_value"`
	HealSources      []SourceEntry `json:"heal_sources,omitempty"`
	GoldSpentWards   int           `json:"gold_spent_wards"`
	GoldSpentSmoke   int           `json:"gold_spent_smoke"`
	GoldSpentDust    int           `json:"gold_spent_dust"`
	GoldLost         int           `json:"gold_lost"`
	CampsStacked     int           `json:"camps_stacked"`
	CreepsStacked    int           `json:"creeps_stacked"`
	RunePickups      int           `json:"rune_pickups"`
	WisdomsCaptured  int           `json:"wisdoms_captured"`
	WatchersCaptured int           `json:"watchers_captured"`
	LotusesGathered  int           `json:"lotuses_gathered"`
	FirstBlood       bool          `json:"first_blood"`
	CourierKills     int           `json:"courier_kills"`
}

type SourceEntry struct {
	Inflictor string  `json:"inflictor"`
	Category  string  `json:"category"`
	Duration  float64 `json:"duration,omitempty"`
	Value     float64 `json:"value,omitempty"`
	Count     int     `json:"count,omitempty"`
}
