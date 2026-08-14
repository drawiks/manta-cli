package stats

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dotabuff/manta"
	"github.com/dotabuff/manta/dota"
	"github.com/klauspost/compress/zstd"
)

const (
	prClass              = "CDOTA_PlayerResource"
	heroClassPrefix      = "CDOTA_Unit_Hero_"
	dataTeamRadiantClass = "CDOTA_DataRadiant"
	dataTeamDireClass    = "CDOTA_DataDire"

	// indexBits is the entity index width used by manta to encode handles.
	indexBits = uint64(14)
)

var matchIDRe = regexp.MustCompile(`^(\d+)`)

// decompressReader wraps r, transparently decompressing zstd and bzip2
// streams. The format is detected from the magic bytes, not the file
// extension: Valve serves replays as zstd while keeping the legacy .dem.bz2
// name. The returned closer releases only the stream wrapper (a zstd decoder)
// and is nil for plain and bzip2 streams.
func decompressReader(r io.Reader) (io.Reader, io.Closer, error) {
	br := bufio.NewReader(r)
	head, err := br.Peek(4)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	var reader io.Reader = br
	var closer io.Closer
	switch {
	case len(head) >= 3 && bytes.Equal(head[:3], []byte("BZh")):
		// bzip2 magic "BZh".
		reader = bzip2.NewReader(br)
	case len(head) >= 4 && bytes.Equal(head[:4], []byte{0x28, 0xb5, 0x2f, 0xfd}):
		// zstd magic.
		zr, err := zstd.NewReader(br, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, nil, err
		}
		reader = zr
		closer = zstdCloser{zr}
	}
	return reader, closer, nil
}

// openReader opens the replay file, transparently decompressing zstd and
// bzip2 streams. Nothing is written to disk.
func OpenReader(path string) (io.Reader, io.Closer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	r, closer, err := decompressReader(f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if closer != nil {
		return r, closeAll{closer, f}, nil
	}
	return r, f, nil
}

// closeAll closes several closers in order, returning the first error.
type closeAll []io.Closer

func (c closeAll) Close() error {
	var first error
	for _, cl := range c {
		if err := cl.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// zstdCloser adapts *zstd.Decoder to io.Closer: its Close returns nothing.
type zstdCloser struct{ *zstd.Decoder }

func (z zstdCloser) Close() error {
	z.Decoder.Close()
	return nil
}

// matchIDFromFilename extracts the leading numeric id from the file name
// (12345_...dem.bz2 -> 12345), or 0 when nothing is found.
func matchIDFromFilename(path string) uint64 {
	m := matchIDRe.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return 0
	}
	id, _ := strconv.ParseUint(m[1], 10, 64)
	return id
}

// mantaPR adapts the CDOTA_PlayerResource entity to the prReader interface.
type mantaPR struct {
	e *manta.Entity
}

// intAny returns the first present value among the given field names. Stat
// fields have moved between m_vecPlayerData and m_vecPlayerTeamData across
// patches (spec §8.7); trying both keeps the reader robust. Fields may also
// flip between int32 and uint32/uint64 encodings on the wire, so each name is
// tried against both widths before moving on.
func (r *mantaPR) intAny(names ...string) int32 {
	for _, n := range names {
		if v, ok := r.e.GetInt32(n); ok {
			return v
		}
	}
	for _, n := range names {
		if v, ok := r.e.GetUint32(n); ok {
			return int32(v)
		}
	}
	for _, n := range names {
		if v, ok := r.e.GetUint64(n); ok {
			return int32(v)
		}
	}
	return 0
}

func (r *mantaPR) slot(i int) (prSlot, bool) {
	base := fmt.Sprintf("m_vecPlayerData.%04d.", i)
	tbase := fmt.Sprintf("m_vecPlayerTeamData.%04d.", i)

	var s prSlot
	steamID, _ := r.e.GetUint64(base + "m_iPlayerSteamID")
	team, _ := int32OrUint32(r.e, base+"m_iPlayerTeam")
	if steamID == 0 || (team != teamRadiant && team != teamDire) {
		return s, false
	}

	s.steamID = steamID
	s.team = team
	s.name, _ = r.e.GetString(base + "m_iszPlayerName")
	s.kills = r.intAny(base+"m_iKills", tbase+"m_iKills")
	s.deaths = r.intAny(base+"m_iDeaths", tbase+"m_iDeaths")
	s.assists = r.intAny(base+"m_iAssists", tbase+"m_iAssists")
	s.level = r.intAny(base+"m_iLevel", tbase+"m_iLevel")
	s.bountyRunes = r.intAny(base+"m_iBountyRunes", tbase+"m_iBountyRunes")
	s.powerRunes = r.intAny(base+"m_iPowerRunes", tbase+"m_iPowerRunes")
	s.waterRunes = r.intAny(base+"m_iWaterRunes", tbase+"m_iWaterRunes")
	s.heroID, _ = int32OrUint32(r.e, tbase+"m_nSelectedHeroID")
	s.heroHandle, _ = r.e.GetUint64(tbase + "m_hSelectedHero")
	s.teamSlot, _ = int32OrUint32(r.e, tbase+"m_iTeamSlot")
	return s, true
}

// handleOf reconstructs the entity handle from index and serial, matching
// manta's FindEntityByHandle encoding.
func handleOf(e *manta.Entity) uint64 {
	return uint64(uint32(e.GetSerial()))<<indexBits | uint64(uint32(e.GetIndex())&((1<<indexBits)-1))
}

// mantaTeams adapts the CDOTA_DataRadiant/CDOTA_DataDire entities to the
// teamReader interface. The camps/creeps stacked counters live under
// m_vecDataTeam.<teamSlot>, each player reading their own row. The
// per-player scoreboard numbers (total earned gold/xp, tower damage, healing,
// net worth, last hits, stuns, wards placed, smokes used) sit in the same
// sub-struct and are the values Valve's match API and OpenDota report, so when
// present they become the authoritative source for GPM/XPM, tower damage and
// the consumable-gold counts. Hero damage is the exception (the scoreboard
// m_flHeroDamage diverges from OpenDota), so it stays on the combat log.
type mantaTeams struct {
	radiant *manta.Entity
	dire    *manta.Entity
}

func (t *mantaTeams) team(side string, teamSlot int32) (teamData, bool) {
	var e *manta.Entity
	switch side {
	case "radiant":
		e = t.radiant
	case "dire":
		e = t.dire
	}
	if e == nil {
		return teamData{}, false
	}
	base := fmt.Sprintf("m_vecDataTeam.%04d.", teamSlot)
	camps, ok := int32OrUint32(e, base+"m_iCampsStacked")
	if !ok {
		return teamData{}, false
	}
	creeps, _ := int32OrUint32(e, base+"m_iCreepsStacked")
	td := teamData{campsStacked: int64(camps), creepsStacked: int64(creeps)}

	// Scoreboard section. m_iTotalEarnedGold and m_flTowerDamage gate the
	// section: when either is absent the patch has no scoreboard numbers, so
	// score stays false and callers keep the combat-log aggregates. The other
	// fields each carry a -1 sentinel: a field missing on the patch keeps the
	// combat-log value instead of overwriting it with a bogus zero.
	td.totalEarnedGold = -1
	td.totalEarnedXP = -1
	td.towerDamage = -1
	td.healing = -1
	td.netWorth = -1
	td.lastHitCount = -1
	td.stuns = -1
	td.runePickups = -1

	gold, ok := int32OrUint32(e, base+"m_iTotalEarnedGold")
	if !ok {
		return td, true
	}
	td.totalEarnedGold = int64(gold)
	if tower, ok := e.GetFloat32(base + "m_flTowerDamage"); ok {
		td.towerDamage = float64(tower)
	} else {
		return td, true
	}
	if xp, ok := int32OrUint32(e, base+"m_iTotalEarnedXP"); ok {
		td.totalEarnedXP = int64(xp)
	}
	if healing, ok := e.GetFloat32(base + "m_fHealing"); ok {
		td.healing = float64(healing)
	}
	if netWorth, ok := int32OrUint32(e, base+"m_iNetWorth"); ok {
		td.netWorth = int64(netWorth)
	}
	if lh, ok := int32OrUint32(e, base+"m_iLastHitCount"); ok {
		td.lastHitCount = int64(lh)
	}
	if stuns, ok := e.GetFloat32(base + "m_fStuns"); ok {
		td.stuns = float64(stuns)
	}
	if runes, ok := int32OrUint32(e, base+"m_iRunePickups"); ok {
		td.runePickups = int64(runes)
	}

	// Ward/smoke counts are the per-player scoreboard numbers used for
	// gold_spent_wards/smoke; -1 marks a patch without the fields. Observer
	// ward placements are free, so only the sentry placement count is read.
	td.sentryWardsPlaced = -1
	td.smokesUsed = -1
	if w, ok := int32OrUint32(e, base+"m_iSentryWardsPlaced"); ok {
		td.sentryWardsPlaced = int64(w)
	}
	if s, ok := int32OrUint32(e, base+"m_iSmokesUsed"); ok {
		td.smokesUsed = int64(s)
	}

	td.score = true
	return td, true
}

func playerIDOf(e *manta.Entity) int32 {
	if id, ok := e.GetInt32("m_iPlayerID"); ok {
		return id
	}
	// The field is serialized as uint32 on newer patches (spec §8.7);
	// GetInt32 would silently return 0 and collapse the pid/xpID binding.
	if id, ok := e.GetUint32("m_iPlayerID"); ok {
		return int32(id)
	}
	return 0
}

// int32OrUint32 reads an int32 field, falling back to the uint32 wire
// encoding used on newer patches (the same flip as m_iPlayerID, spec §8.7).
func int32OrUint32(e *manta.Entity, name string) (int32, bool) {
	if v, ok := e.GetInt32(name); ok {
		return v, true
	}
	if v, ok := e.GetUint32(name); ok {
		return int32(v), true
	}
	return 0, false
}

// parseReplay parses a single replay file and returns the aggregated result.
// When no match id can be derived from the filename it falls back to the
// replay's own CDemoFileInfo metadata.
func ParseReplay(path string) (*Result, error) {
	r, closer, err := OpenReader(path)
	if err != nil {
		return nil, err
	}
	return parseStream(r, closer, matchIDFromFilename(path))
}

// parseStdin parses a replay stream read from r. The format is detected from
// the magic bytes; the match id can only come from the replay's CDemoFileInfo
// metadata.
func ParseStdin(r io.Reader) (*Result, error) {
	reader, closer, err := decompressReader(r)
	if err != nil {
		return nil, err
	}
	return parseStream(reader, closer, 0)
}

// parseStream parses a replay stream and returns the aggregated result.
func parseStream(r io.Reader, closer io.Closer, matchID uint64) (*Result, error) {
	if closer != nil {
		defer closer.Close()
	}

	p, err := manta.NewStreamParser(r)
	if err != nil {
		return nil, err
	}

	aggr := newAggregator()
	aggr.lookup = p.LookupStringByIndex
	var pr *manta.Entity
	teams := &mantaTeams{}
	heroes := make(map[uint64]heroInfo)

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		switch {
		case e.GetClassName() == prClass:
			if op.Flag(manta.EntityOpEntered) {
				pr = e
			}
		case e.GetClassName() == dataTeamRadiantClass:
			if op.Flag(manta.EntityOpEntered) {
				teams.radiant = e
			}
		case e.GetClassName() == dataTeamDireClass:
			if op.Flag(manta.EntityOpEntered) {
				teams.dire = e
			}
		case strings.HasPrefix(e.GetClassName(), heroClassPrefix):
			// Track the scheduled respawn clock on every update (death is a
			// delta, not a create); the hero identity is bound once on enter.
			// m_lifeState is never transmitted on modern patches; the hero
			// entity reports m_flRespawnTime = death time + respawn duration
			// on death instead, which pairs with the combat-log DEATH clocks
			// to compute time_dead (spec §6.5).
			if respawn, ok := e.GetFloat32("m_flRespawnTime"); ok {
				aggr.respawnUpdate(handleOf(e), respawn)
			}
			if op.Flag(manta.EntityOpEntered) {
				heroes[handleOf(e)] = heroInfo{
					name:     strings.TrimPrefix(e.GetClassName(), heroClassPrefix),
					playerID: playerIDOf(e),
				}
			}
		}
		return nil
	})

	p.Callbacks.OnCMsgDOTACombatLogEntry(func(e *dota.CMsgDOTACombatLogEntry) error {
		aggr.handleCombatLog(e, p.Tick)
		return nil
	})

	p.Callbacks.OnCDemoFileInfo(func(info *dota.CDemoFileInfo) error {
		if matchID == 0 {
			if d := info.GetGameInfo().GetDota(); d != nil {
				matchID = d.GetMatchId()
			}
		}
		return nil
	})

	if err := p.Start(); err != nil {
		return nil, err
	}

	if pr == nil {
		return nil, fmt.Errorf("CDOTA_PlayerResource not found")
	}

	return buildResult(aggr, &mantaPR{e: pr}, heroes, teams, matchID), nil
}
