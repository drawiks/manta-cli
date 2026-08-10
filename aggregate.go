package main

import (
	"math"
	"strconv"

	"github.com/dotabuff/manta/dota"
)

const (
	teamRadiant = 2
	teamDire    = 3

	// CMsgDOTACombatLogEntry GAME_STATE values use the DOTA_GameState enum
	// (dota_shared_enums.pb.go), not DOTA_GAMERULES_STATE.
	gameInProgressState = 5
	postGameState       = 6

	// Current-patch prices used to convert ITEM consumable uses/placements and
	// scoreboard purchase counts into gold (spec §6.4). The replay itself does
	// not carry per-item gold (PURCHASE events lost the item id on modern
	// patches), so the prices are constants. Gem of True Sight is never
	// observable in a replay (its item id does not appear in the combat log on
	// modern patches), so gold_spent_wards counts it as 0.
	itemWardPrice  = 50 // sentry ward; observer wards are free on current patches
	itemSmokePrice = 50
	itemDustPrice  = 80
)

// xpStats accumulates combat-log aggregates in the xpID id-space (spec §6.3).
type xpStats struct {
	towerDamage  int64
	healing      float64
	stunDuration float64
	heroDamage   int64
	damageTaken  int64
	goldTotal    int64
	xpTotal      int64
	goldWards    int64
	goldSmoke    int64
	goldDust     int64

	// deathClocks holds the game-clock timestamps of the player's hero deaths
	// (spec §6.5), paired by index with the hero entity's m_flRespawnTime
	// values to compute time_dead.
	deathClocks []float32
}

// slotStats binds a PlayerResource slot index (0..9, the PLAYERSTATS target)
// to its xpID and stores the max last_hits/networth reported by PLAYERSTATS.
type slotStats struct {
	xpID     uint32
	lastHits uint32
	networth uint32
}

// aggregator processes the combat-log event stream (spec §6.4).
type aggregator struct {
	bySlot map[uint32]*slotStats
	byXp   map[uint32]*xpStats

	gameInProgress float32
	postGame       float32
	lastTbkillTeam uint32

	// lookup resolves combat-log name indexes (InflictorName etc.) against the
	// "CombatLogNames" string table, e.g. "item_dust" for PURCHASE events. It
	// is injected from the parser; tests substitute a fake resolver.
	lookup func(table string, index int32) (string, bool)

	// Dead-time tracking (hero m_flRespawnTime transitions, spec §6.5).
	// respawnVals holds the scheduled respawn clocks the hero entity reports
	// on death (m_flRespawnTime = death time + respawn duration). Each entry
	// pairs by index with xpStats.deathClocks: time_dead is the sum of
	// (min(respawn, buyback, post-game) - death). m_flRespawnTime is the
	// scheduled value and never resets on buyback, so buybackClocks caps the
	// matching window, and the post-game clock caps heroes still dead at the
	// end. lastRespawn/seenRespawn track the field's changes across updates.
	respawnVals   map[uint64][]float32
	lastRespawn   map[uint64]float32
	seenRespawn   map[uint64]bool
	buybackClocks map[uint32][]float32

	// First blood is the FIRST_BLOOD (type 18) combat-log event (spec §6.4).
	// That event carries no attacker index, so the credit is taken from the
	// first hero DEATH at/after its tick; the plain first hero DEATH is the
	// fallback for replays without a type-18 event. The attacker xpIDs are
	// tracked with separate presence flags (not a 0 sentinel): xpID 0 is a
	// valid id (a PLAYERSTATS value, spec §6.3), so 0 can be the FB attacker.
	fb18Seen    bool
	fb18Tick    uint32
	fbFirstHero uint32
	fbHasFirst  bool
	fbAfter18   uint32
	fbHasAfter  bool
	fbXP        uint32
	fbResolved  bool
}

func newAggregator() *aggregator {
	return &aggregator{
		bySlot:        make(map[uint32]*slotStats),
		byXp:          make(map[uint32]*xpStats),
		respawnVals:   make(map[uint64][]float32),
		lastRespawn:   make(map[uint64]float32),
		seenRespawn:   make(map[uint64]bool),
		buybackClocks: make(map[uint32][]float32),
	}
}

func (a *aggregator) xp(id uint32) *xpStats {
	x, ok := a.byXp[id]
	if !ok {
		x = &xpStats{}
		a.byXp[id] = x
	}
	return x
}

func (a *aggregator) slot(id uint32) *slotStats {
	p, ok := a.bySlot[id]
	if !ok {
		p = &slotStats{}
		a.bySlot[id] = p
	}
	return p
}

// handleCombatLog dispatches a single combat-log entry (spec §6.4).
func (a *aggregator) handleCombatLog(e *dota.CMsgDOTACombatLogEntry, tick uint32) {
	switch e.GetType() {
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE:
		// The combat-log GAME_STATE value uses the DOTA_GameState enum. The
		// event's Timestamp is the game clock, which freezes during pauses,
		// so POST_GAME - GAME_IN_PROGRESS is the play time OpenDota reports.
		switch e.GetValue() {
		case gameInProgressState:
			a.gameInProgress = e.GetTimestamp()
		case postGameState:
			a.postGame = e.GetTimestamp()
		}
		return

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE:
		if e.GetIsTargetBuilding() {
			a.xp(e.GetAttackerName()).towerDamage += int64(e.GetValue())
		}
		if e.GetIsTargetHero() && !e.GetIsTargetIllusion() && e.GetAttackerName() != e.GetTargetName() {
			// Damage dealt to a hero (self-damage and damage to illusions
			// excluded, mirroring the end-game summary and OpenDota).
			a.xp(e.GetAttackerName()).heroDamage += int64(e.GetValue())
		}
		if e.GetIsTargetHero() && !e.GetIsTargetIllusion() && e.GetIsAttackerHero() {
			// Damage taken from heroes, including self-inflicted damage (the
			// game counts it, and OpenDota's hero-source sum includes it).
			a.xp(e.GetTargetName()).damageTaken += int64(e.GetValue())
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_ITEM:
		// ITEM events carry the item's id in inflictor_name; PURCHASE events
		// stopped carrying it on modern patches. They log item uses/placements
		// rather than purchases, so the gold is derived from current-patch
		// prices. When the scoreboard exposes m_iSentryWardsPlaced/m_iSmokesUsed,
		// buildPlayer prefers those counts over these sums (spec §6.4). Observer
		// ward placements are free and never contribute to gold_spent_wards.
		name, ok := a.itemName(e.GetInflictorName())
		if !ok {
			return
		}
		x := a.xp(e.GetAttackerName())
		switch name {
		case "item_ward_sentry":
			x.goldWards += itemWardPrice
		case "item_smoke_of_deceit":
			x.goldSmoke += itemSmokePrice
		case "item_dust":
			x.goldDust += itemDustPrice
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL:
		if e.GetIsTargetHero() {
			a.xp(e.GetAttackerName()).healing += float64(e.GetValue())
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD:
		if stun := e.GetStunDuration(); stun > 0 {
			a.xp(e.GetAttackerName()).stunDuration += float64(stun)
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH:
		if e.GetIsTargetHero() {
			if !a.fbHasFirst {
				a.fbHasFirst = true
				a.fbFirstHero = e.GetAttackerName()
			}
			if a.fb18Seen && tick >= a.fb18Tick && !a.fbHasAfter {
				a.fbHasAfter = true
				a.fbAfter18 = e.GetAttackerName()
			}
			a.xp(e.GetTargetName()).deathClocks = append(
				a.xp(e.GetTargetName()).deathClocks, e.GetTimestamp())
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_BUYBACK:
		// value is the PlayerResource slot index (0..9), same space as the
		// PLAYERSTATS target; the timestamps accumulate per slot in
		// chronological order, each capping the matching dead window.
		a.buybackClocks[e.GetValue()] = append(a.buybackClocks[e.GetValue()], e.GetTimestamp())

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_FIRST_BLOOD:
		if !a.fb18Seen {
			a.fb18Seen = true
			a.fb18Tick = tick
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD:
		// value is signed int32 (gold lost on death arrives as a negative
		// delta); the protobuf field is uint32, so cast back before summing.
		a.xp(e.GetTargetName()).goldTotal += int64(int32(e.GetValue()))

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_XP:
		a.xp(e.GetTargetName()).xpTotal += int64(int32(e.GetValue()))

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PLAYERSTATS:
		// target is the PlayerResource slot index (0..9) on every patch
		// (spec §6.3); value is the xpID used in the other event id-spaces.
		s := a.slot(e.GetTargetName())
		s.xpID = e.GetValue()
		if e.GetLastHits() > s.lastHits {
			s.lastHits = e.GetLastHits()
		}
		if e.GetNetworth() > s.networth {
			s.networth = e.GetNetworth()
		}

	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_TEAM_BUILDING_KILL:
		a.lastTbkillTeam = e.GetTargetTeam()
	}
}

// itemName resolves a combat-log name index (PURCHASE inflictor_name) against
// the "CombatLogNames" string table.
func (a *aggregator) itemName(index uint32) (string, bool) {
	if a.lookup == nil {
		return "", false
	}
	return a.lookup("CombatLogNames", int32(index))
}

// respawnUpdate tracks a hero's scheduled respawn clock from its
// m_flRespawnTime field on every entity update. The field is negative while
// alive and jumps to death time + respawn duration on death; it is the
// scheduled value, so buybacks leave it untouched. The first observation is
// the initial state and never recorded; later positive changes append a value
// that pairs by index with the player's death clocks.
func (a *aggregator) respawnUpdate(handle uint64, respawn float32) {
	if !a.seenRespawn[handle] {
		a.seenRespawn[handle] = true
		a.lastRespawn[handle] = respawn
		return
	}
	if respawn == a.lastRespawn[handle] {
		return
	}
	a.lastRespawn[handle] = respawn
	if respawn > 0 {
		a.respawnVals[handle] = append(a.respawnVals[handle], respawn)
	}
}

// timeDead returns the total time a hero spent dead: the sum of
// min(respawn, buyback, post-game) - death over the player's deaths, where
// respawn is the scheduled respawn clock, buyback the earliest buyback after
// the death (multiple buybacks per slot apply to successive windows), and
// post-game the clock the match stops (a hero still dead at the end is
// counted only until then; when the post-game clock is unknown, the window is
// capped by the respawn alone). All clocks share the combat-log timeline, so
// pauses need no special handling.
func (a *aggregator) timeDead(handle uint64, xpID, slot uint32) float64 {
	deaths := a.xpStats(xpID).deathClocks
	respawns := a.respawnVals[handle]
	buybacks := a.buybackClocks[slot]
	var total float64
	for i, r := range respawns {
		if i >= len(deaths) {
			break
		}
		end := r
		for _, bb := range buybacks {
			if bb > deaths[i] {
				if bb < end {
					end = bb
				}
				break
			}
		}
		if a.postGame > 0 && end > a.postGame {
			end = a.postGame
		}
		if d := float64(end - deaths[i]); d > 0 {
			total += d
		}
	}
	return total
}

// durationSec returns the game duration in seconds, matching OpenDota: the
// difference of the POST_GAME and GAME_IN_PROGRESS game-clock timestamps
// (the clock freezes during pauses), truncated to whole seconds. Returns 0
// if either state is absent.
func (a *aggregator) durationSec() int32 {
	if a.gameInProgress == 0 || a.postGame == 0 || a.postGame <= a.gameInProgress {
		return 0
	}
	return int32(a.postGame - a.gameInProgress)
}

// durationFloat returns the same duration as an un-truncated float, which is
// what the game itself divides against when computing the per-minute stats.
func (a *aggregator) durationFloat() float64 {
	if a.gameInProgress == 0 || a.postGame == 0 || a.postGame <= a.gameInProgress {
		return 0
	}
	return float64(a.postGame - a.gameInProgress)
}

// radiantWin reports the winner via the last destroyed building (spec §6.7).
func (a *aggregator) radiantWin() bool {
	return a.lastTbkillTeam == teamDire
}

// resolveFB pins the first-blood attacker xpID once, preferring the hero death
// that corresponds to the FIRST_BLOOD (type 18) event over the plain first
// hero death.
func (a *aggregator) resolveFB() {
	if a.fbResolved {
		return
	}
	if a.fb18Seen && a.fbHasAfter {
		a.fbXP = a.fbAfter18
	} else {
		a.fbXP = a.fbFirstHero
	}
	a.fbResolved = true
}

// prSlot is the per-slot snapshot taken from CDOTA_PlayerResource.
type prSlot struct {
	steamID     uint64
	team        int32
	teamSlot    int32
	name        string
	kills       int32
	deaths      int32
	assists     int32
	level       int32
	bountyRunes int32
	powerRunes  int32
	waterRunes  int32
	heroID      int32
	heroHandle  uint64
}

// prReader abstracts the PlayerResource entity so assembly is testable.
type prReader interface {
	slot(i int) (prSlot, bool)
}

// heroInfo is resolved from the CDOTA_Unit_Hero_ entity.
type heroInfo struct {
	name     string
	playerID int32
}

// teamData is the per-team snapshot taken from the CDOTA_DataRadiant/Dire
// entities. CampsStacked and CreepsStacked are team stats, so every player of
// a team reports the same value, mirroring OpenDota. The remaining fields are
// the same per-player scoreboard numbers Valve's match API (and OpenDota)
// exposes (gold_per_min, xp_per_min, tower_damage, ...); score reports whether
// the scoreboard section was readable so callers can fall back to the
// combat-log aggregates on patches without it. Each per-player scoreboard
// field uses -1 as its "absent on this patch" sentinel, so a field missing
// on a future patch keeps the combat-log value instead of a bogus zero
// (m_iTotalEarnedGold and m_flTowerDamage gate score, so those two are never
// -1 when score is set).
type teamData struct {
	campsStacked  int64
	creepsStacked int64

	totalEarnedGold int64
	totalEarnedXP   int64
	towerDamage     float64
	healing         float64
	netWorth        int64
	lastHitCount    int64
	stuns           float64
	runePickups     int64

	sentryWardsPlaced int64
	smokesUsed        int64

	score bool
}

// teamReader abstracts the team-data entities so assembly is testable.
type teamReader interface {
	team(side string, teamSlot int32) (teamData, bool)
}

func teamName(team int32) string {
	switch team {
	case teamRadiant:
		return "radiant"
	case teamDire:
		return "dire"
	default:
		return ""
	}
}

// buildResult assembles the final Result from the aggregated state.
func buildResult(a *aggregator, pr prReader, heroes map[uint64]heroInfo, teams teamReader, matchID uint64) *Result {
	a.resolveFB()
	dur := a.durationSec()
	durFloat := a.durationFloat()
	res := &Result{
		MatchID:     matchID,
		DurationSec: dur,
		RadiantWin:  a.radiantWin(),
		Players:     make([]Player, 0, 10),
	}
	for i := 0; i < 10; i++ {
		slot, ok := pr.slot(i)
		if !ok {
			continue
		}
		res.Players = append(res.Players, buildPlayer(a, slot, heroes, teams, i, durFloat))
	}
	return res
}

func buildPlayer(a *aggregator, slot prSlot, heroes map[uint64]heroInfo, teams teamReader, slotIdx int, durFloat float64) Player {
	// player_id mirrors the hero entity's m_iPlayerID for reference only
	// (even 0..18 on current patches, raw slot 0..9 on legacy patches); the
	// combat-log binding always keys on the slot index (spec §6.3).
	playerID := int32(2 * slotIdx)
	if h, ok := heroes[slot.heroHandle]; ok {
		playerID = h.playerID
	}

	ss := a.slot(uint32(slotIdx))
	xp := a.xpStats(ss.xpID)

	p := Player{
		SteamID:      slot.steamID,
		PlayerID:     playerID,
		HeroID:       slot.heroID,
		Hero:         heroes[slot.heroHandle].name,
		Team:         teamName(slot.team),
		Name:         slot.name,
		Kills:        slot.kills,
		Deaths:       slot.deaths,
		Assists:      slot.assists,
		Level:        slot.level,
		LastHits:     ss.lastHits,
		Networth:     ss.networth,
		Healing:      shortestFloat32(xp.healing),
		HeroDamage:   xp.heroDamage,
		DamageTaken:  xp.damageTaken,
		TowerDamage:  xp.towerDamage,
		TimeDead:     shortestFloat32(a.timeDead(slot.heroHandle, ss.xpID, uint32(slotIdx))),
		StunDuration: xp.stunDuration,
		GoldWards:    xp.goldWards,
		GoldSmoke:    xp.goldSmoke,
		GoldDust:     xp.goldDust,
		RunePickups:  slot.bountyRunes + slot.powerRunes + slot.waterRunes,
		// FirstBlood is only claimed when a FB candidate exists (a hero death,
		// or a hero death at/after the type-18 event). The xpID 0 guard is not
		// used: xpID 0 is a valid id (spec §6.3), so it must be claimable.
		FirstBlood: a.fbResolved && (a.fbHasFirst || a.fbHasAfter) && a.fbXP == ss.xpID,
	}

	var td teamData
	var haveTD bool
	if teams != nil {
		if t, ok := teams.team(teamName(slot.team), slot.teamSlot); ok {
			td, haveTD = t, true
		}
	}

	if haveTD {
		p.CampsStacked = td.campsStacked
		p.CreepsStacked = td.creepsStacked
		if td.score {
			// The game's own scoreboard numbers are what Valve's match API
			// (and thus OpenDota) reports for GPM/XPM, tower damage, healing,
			// stuns and net worth; prefer them over the combat-log sums when
			// the entities expose them. Hero damage is the exception: the
			// scoreboard m_flHeroDamage diverges from OpenDota's hero damage
			// (e.g. +9k for Pudge on the reference match), so hero_damage
			// stays on the combat-log sum. Every field is guarded by its -1
			// absent-sentinel: a field missing on the patch keeps the
			// combat-log value instead of overwriting it with a bogus zero.
			if td.lastHitCount >= 0 {
				p.LastHits = uint32(td.lastHitCount)
			}
			if td.netWorth >= 0 {
				p.Networth = uint32(td.netWorth)
			}
			if td.healing >= 0 {
				p.Healing = shortestFloat32(td.healing)
			}
			if td.towerDamage >= 0 {
				p.TowerDamage = int64(td.towerDamage)
			}
			if td.stuns >= 0 {
				p.StunDuration = shortestFloat32(td.stuns)
			}
			if td.runePickups >= 0 {
				p.RunePickups = int32(td.runePickups)
			}
			if td.sentryWardsPlaced >= 0 {
				p.GoldWards = td.sentryWardsPlaced * itemWardPrice
			}
			if td.smokesUsed >= 0 {
				p.GoldSmoke = td.smokesUsed * itemSmokePrice
			}
		}
	}

	if durFloat > 0 {
		// The combat-log GPM/XPM are the fallback; the scoreboard numbers
		// override them per field (never with a bogus zero for a missing
		// field, thanks to the -1 sentinel). When the scoreboard is used the
		// divisor is the un-truncated game clock (e.g. 2463.5334), which is
		// what the match API reports per-player (not duration_sec's
		// truncation), and the value is floored exactly like the game.
		minutes := durFloat / 60.0
		gpm := uint32(math.Round(float64(xp.goldTotal) / minutes))
		xpm := uint32(math.Round(float64(xp.xpTotal) / minutes))
		if haveTD && td.score {
			if td.totalEarnedGold >= 0 {
				gpm = uint32(math.Floor(float64(td.totalEarnedGold) * 60 / durFloat))
			}
			if td.totalEarnedXP >= 0 {
				xpm = uint32(math.Floor(float64(td.totalEarnedXP) * 60 / durFloat))
			}
		}
		p.GPM = gpm
		p.XPM = xpm
	}
	return p
}

// shortestFloat32 collapses a float64 that originated from a float32 field to
// the shortest decimal that round-trips through float32, mirroring how the
// match API serializes scoreboard floats (e.g. stuns 12.433472, not
// 12.4334716796875).
func shortestFloat32(v float64) float64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', -1, 32), 64)
	return f
}

func (a *aggregator) xpStats(id uint32) xpStats {
	if x, ok := a.byXp[id]; ok {
		return *x
	}
	return xpStats{}
}
