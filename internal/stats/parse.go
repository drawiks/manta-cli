package stats

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"fmt"
	"io"
	"math"
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
	rulesClass           = "CDOTAGamerulesProxy"

	teamRadiant = 2
	teamDire    = 3
)

var matchIDRe = regexp.MustCompile(`^(\d+)`)

// decompressReader wraps r, transparently decompressing zstd and bzip2
// streams. The format is detected from magic bytes, not the file extension.
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
		reader = bzip2.NewReader(br)
	case len(head) >= 4 && bytes.Equal(head[:4], []byte{0x28, 0xb5, 0x2f, 0xfd}):
		zr, err := zstd.NewReader(br, zstd.WithDecoderConcurrency(1))
		if err != nil {
			return nil, nil, err
		}
		reader = zr
		closer = zstdCloser{zr}
	}
	return reader, closer, nil
}

// OpenReader opens path, transparently decompressing zstd/bzip2.
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

type zstdCloser struct{ *zstd.Decoder }

func (z zstdCloser) Close() error {
	z.Decoder.Close()
	return nil
}

func matchIDFromFilename(path string) int64 {
	m := matchIDRe.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return 0
	}
	id, _ := strconv.ParseInt(m[1], 10, 64)
	return id
}

func handleOf(e *manta.Entity) uint64 {
	return uint64(uint32(e.GetSerial()))<<14 | uint64(uint32(e.GetIndex())&((1<<14)-1))
}

type slotInfo struct {
	slot       int
	prIndex    int
	team       string
	teamSlot   int
	steamID    int64
	name       string
	heroID     int
	heroHandle uint64
	hero       string
}

type pendingCombat struct {
	rawTime int
	ev      combatEvent
}

type chatEvent struct {
	kind    string
	player1 int
	rawTime int
}

type replayState struct {
	p  *manta.Parser
	ag *aggregator

	pr    *manta.Entity
	rules *manta.Entity
	teams map[string]*manta.Entity

	slots      [10]slotInfo
	handleSlot map[uint64]int
	init       bool

	startKnown bool
	gameStart  int
	postGame   bool
	serverTick uint32

	pending          []pendingCombat
	pendingChat      []chatEvent
	pendingIntervals []intervalData

	lastEmitted int
	hasEmitted  bool

	startingItemsWritten [10]bool
	abilityNames         map[uint64]string

	lastCombatTime int
	lastTBKillTeam int
	matchID        int64
}

func ParseReplay(path string) (*Result, error) {
	r, closer, err := OpenReader(path)
	if err != nil {
		return nil, err
	}
	return parseStream(r, closer, matchIDFromFilename(path))
}

func ParseStdin(r io.Reader) (*Result, error) {
	reader, closer, err := decompressReader(r)
	if err != nil {
		return nil, err
	}
	return parseStream(reader, closer, 0)
}

func parseStream(r io.Reader, closer io.Closer, fallbackMatchID int64) (*Result, error) {
	if closer != nil {
		defer closer.Close()
	}

	p, err := manta.NewStreamParser(r)
	if err != nil {
		return nil, err
	}

	st := &replayState{
		p:            p,
		ag:           newAggregator(),
		teams:        map[string]*manta.Entity{},
		handleSlot:   map[uint64]int{},
		abilityNames: map[uint64]string{},
		matchID:      fallbackMatchID,
	}

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		if !op.Flag(manta.EntityOpEntered) {
			return nil
		}
		cn := e.GetClassName()
		switch {
		case cn == prClass:
			st.pr = e
		case cn == rulesClass:
			st.rules = e
		case cn == dataTeamRadiantClass:
			st.teams["radiant"] = e
		case cn == dataTeamDireClass:
			st.teams["dire"] = e
		case strings.HasPrefix(cn, heroClassPrefix):
			if st.resolveDue() {
				st.assignHero(handleOf(e), unitToHeroName(cn))
			}
		}
		st.resolveSlots()
		return nil
	})

	p.Callbacks.OnCNETMsg_Tick(func(m *dota.CNETMsg_Tick) error {
		st.serverTick = m.GetTick()
		st.maybeSample()
		return nil
	})

	p.Callbacks.OnCDemoFileInfo(func(info *dota.CDemoFileInfo) error {
		d := info.GetGameInfo().GetDota()
		if d == nil {
			return nil
		}
		if d.GetMatchId() != 0 {
			st.matchID = int64(d.GetMatchId())
		}
		players := make([]playerInfo, 0, len(d.GetPlayerInfo()))
		for i, pi := range d.GetPlayerInfo() {
			team := "radiant"
			if pi.GetGameTeam() == teamDire {
				team = "dire"
			}
			players = append(players, playerInfo{
				Slot:    i,
				SteamID: int64(pi.GetSteamid()),
				Hero:    pi.GetHeroName(),
				Name:    pi.GetPlayerName(),
				Team:    team,
			})
		}
		st.ag.setEpilogue(
			st.matchID,
			shortestFloat32(float64(info.GetPlaybackTime())),
			d.GetGameWinner() == 2,
			players,
		)
		return nil
	})

	p.Callbacks.OnCMsgDOTACombatLogEntry(func(e *dota.CMsgDOTACombatLogEntry) error {
		st.handleCombat(e)
		return nil
	})

	p.Callbacks.OnCDOTAUserMsg_ChatEvent(func(e *dota.CDOTAUserMsg_ChatEvent) error {
		st.handleChat(e)
		return nil
	})

	if err := p.Start(); err != nil {
		return nil, err
	}

	if st.pr == nil {
		return nil, fmt.Errorf("CDOTA_PlayerResource not found")
	}
	if !st.startKnown {
		st.startKnown = true
		st.gameStart = 0
		st.flushIntervals()
		st.flushPending()
	}
	st.resolveSlots()
	if st.init {
		st.maybeSample()
	}
	st.finishRoster()
	return st.ag.result(), nil
}

func (st *replayState) resolveDue() bool { return !st.init }

func (st *replayState) resolveSlots() {
	if st.init || st.pr == nil {
		return
	}
	added := 0
	for i := 0; i < 30 && added < 10; i++ {
		team, ok := numInt(st.pr, fmt.Sprintf("m_vecPlayerData.%04d.m_iPlayerTeam", i))
		if !ok {
			continue
		}
		if team == 14 {
			return
		}
		if team != teamRadiant && team != teamDire {
			continue
		}
		teamSlot, _ := numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iTeamSlot", i))
		steamID, _ := numUint64(st.pr, fmt.Sprintf("m_vecPlayerData.%04d.m_iPlayerSteamID", i))
		name, _ := st.pr.GetString(fmt.Sprintf("m_vecPlayerData.%04d.m_iszPlayerName", i))
		heroID, _ := numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_nSelectedHeroID", i))
		handle, _ := numUint64(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_hSelectedHero", i))
		teamName := "radiant"
		if team == teamDire {
			teamName = "dire"
		}
		st.slots[added] = slotInfo{
			slot:       added,
			prIndex:    i,
			team:       teamName,
			teamSlot:   teamSlot,
			steamID:    int64(steamID),
			name:       name,
			heroID:     heroID,
			heroHandle: handle,
		}
		hero := st.heroName(handle)
		st.slots[added].hero = hero
		if handle != 0 && handle != 0xFFFFFF {
			st.handleSlot[handle] = added
			st.assignHero(handle, hero)
		}
		st.ag.setSlot(added, teamName, hero)
		added++
	}
	if added == 10 {
		st.init = true
	}
}

func (st *replayState) heroName(handle uint64) string {
	if handle == 0 || handle == 0xFFFFFF {
		return ""
	}
	if e := st.p.FindEntityByHandle(handle); e != nil {
		return unitToHeroName(e.GetClassName())
	}
	return ""
}

func (st *replayState) assignHero(handle uint64, hero string) {
	if hero == "" {
		return
	}
	slot, ok := st.handleSlot[handle]
	if !ok {
		return
	}
	st.ag.setSlot(slot, st.slots[slot].team, hero)
}

func (st *replayState) handleCombat(e *dota.CMsgDOTACombatLogEntry) {
	rawTime := int(math.Round(float64(e.GetTimestamp())))
	st.lastCombatTime = rawTime
	if e.GetType() == dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GAME_STATE {
		switch e.GetValue() {
		case 5:
			if !st.startKnown {
				st.gameStart = rawTime
				st.startKnown = true
				st.flushIntervals()
				st.flushPending()
			}
		case 6:
			st.maybeSample()
			st.postGame = true
		}
		return
	}
	if e.GetType() == dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_TEAM_BUILDING_KILL {
		st.lastTBKillTeam = int(e.GetTargetTeam())
	}
	ev := st.combatEvent(e)
	if !st.startKnown {
		st.pending = append(st.pending, pendingCombat{rawTime: rawTime, ev: ev})
		return
	}
	ev.Time = rawTime - st.gameStart
	st.dispatchCombat(ev)
}

func (st *replayState) combatEvent(e *dota.CMsgDOTACombatLogEntry) combatEvent {
	return combatEvent{
		Type:             e.GetType().String(),
		AttackerName:     st.combatName(e.GetAttackerName()),
		TargetName:       st.combatName(e.GetTargetName()),
		SourceName:       st.combatName(e.GetDamageSourceName()),
		TargetSourceName: st.combatName(e.GetTargetSourceName()),
		Inflictor:        st.combatName(e.GetInflictorName()),
		ValueName:        st.purchaseName(e),
		AttackerHero:     e.GetIsAttackerHero(),
		TargetHero:       e.GetIsTargetHero(),
		AttackerIllusion: e.GetIsAttackerIllusion(),
		TargetIllusion:   e.GetIsTargetIllusion(),
		Value:            int(int32(e.GetValue())),
		StunDuration:     float64(e.GetStunDuration()),
		GoldReason:       int(e.GetGoldReason()),
	}
}

func (st *replayState) combatName(idx uint32) string {
	if idx == 0 {
		return "dota_unknown"
	}
	if s, ok := st.p.LookupStringByIndex("CombatLogNames", int32(idx)); ok && s != "" {
		return s
	}
	return "dota_unknown"
}

func (st *replayState) purchaseName(e *dota.CMsgDOTACombatLogEntry) string {
	if e.GetType() != dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PURCHASE {
		return ""
	}
	if s, ok := st.p.LookupStringByIndex("CombatLogNames", int32(e.GetValue())); ok {
		return s
	}
	return ""
}

func (st *replayState) dispatchCombat(ev combatEvent) {
	switch ev.Type {
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DAMAGE.String():
		st.ag.onDamage(ev)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_HEAL.String():
		st.ag.onHeal(ev)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD.String():
		st.ag.onModifier(ev, true)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE.String():
		st.ag.onModifier(ev, false)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_PURCHASE.String():
		st.ag.onPurchase(ev)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_GOLD.String():
		st.ag.onGold(ev)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_DEATH.String():
		st.ag.onDeath(ev)
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_ABILITY.String():
		st.ag.onAbility(ev)
	}
}

func (st *replayState) handleChat(e *dota.CDOTAUserMsg_ChatEvent) {
	kind := e.GetType().String()
	if kind != "CHAT_MESSAGE_FIRSTBLOOD" && kind != "CHAT_MESSAGE_MINIBOSS_KILL" {
		return
	}
	ce := chatEvent{
		kind:    kind,
		player1: int(e.GetPlayerid_1()),
		rawTime: int(math.Round(float64(e.GetTime()))),
	}
	if !st.startKnown {
		st.pendingChat = append(st.pendingChat, ce)
		return
	}
	st.dispatchChat(ce)
}

func (st *replayState) dispatchChat(ce chatEvent) {
	switch ce.kind {
	case "CHAT_MESSAGE_FIRSTBLOOD":
		st.ag.onFirstBlood(ce.player1)
	case "CHAT_MESSAGE_MINIBOSS_KILL":
		t := ce.rawTime - st.gameStart
		if t <= 0 {
			t = st.currentTime() - st.gameStart
		}
		st.ag.onMiniboss(ce.player1, t)
	}
}

func (st *replayState) flushPending() {
	if !st.startKnown {
		return
	}
	for _, p := range st.pending {
		ev := p.ev
		ev.Time = p.rawTime - st.gameStart
		st.dispatchCombat(ev)
	}
	st.pending = nil
	for _, ce := range st.pendingChat {
		st.dispatchChat(ce)
	}
	st.pendingChat = nil
}

func (st *replayState) currentTime() int {
	if st.rules != nil {
		if v, ok := numFloat(st.rules, "m_pGameRules.m_fGameTime"); ok {
			return int(math.Round(v))
		}
		tick := int(st.serverTick)
		if paused, ok := st.rules.GetBool("m_pGameRules.m_bGamePaused"); ok && paused {
			if v, ok := numInt(st.rules, "m_pGameRules.m_nPauseStartTick"); ok {
				tick = v
			}
		}
		pausedTicks, _ := numInt(st.rules, "m_pGameRules.m_nTotalPausedTicks")
		return int(math.Round(float64(tick-pausedTicks) / 30.0))
	}
	return st.lastCombatTime
}

func (st *replayState) maybeSample() {
	if st.postGame || !st.init {
		return
	}
	if !st.startKnown && st.rules != nil {
		if v, ok := numFloat(st.rules, "m_pGameRules.m_flGameStartTime"); ok && v != 0 {
			st.gameStart = int(math.Round(v))
			st.startKnown = true
			st.flushIntervals()
		}
	}
	raw := st.currentTime()
	if st.hasEmitted && raw <= st.lastEmitted {
		return
	}
	st.lastEmitted = raw
	st.hasEmitted = true
	st.snapshotInterval(raw)
}

func (st *replayState) snapshotInterval(raw int) {
	ivs := make([]intervalData, 0, 10)
	for slot := 0; slot < 10; slot++ {
		si := st.slots[slot]
		if h, ok := numUint64(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_hSelectedHero", si.prIndex)); ok && h != 0 && h != 0xFFFFFF {
			si.heroHandle = h
			st.slots[slot].heroHandle = h
			st.handleSlot[h] = slot
		}
		if id, ok := numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_nSelectedHeroID", si.prIndex)); ok {
			si.heroID = id
			st.slots[slot].heroID = id
		}
		hero := st.p.FindEntityByHandle(si.heroHandle)
		var iv intervalData
		iv.Slot = slot
		iv.Time = raw
		iv.HeroID = si.heroID
		if hero != nil {
			iv.Unit = unitToHeroName(hero.GetClassName())
			if v, ok := numInt(hero, "m_lifeState"); ok {
				iv.LifeState = v
				iv.HasLifeState = true
			}
			if cx, ok := numInt(hero, "CBodyComponent.m_cellX"); ok {
				vx, _ := numFloat(hero, "CBodyComponent.m_vecX")
				cy, okY := numInt(hero, "CBodyComponent.m_cellY")
				vy, _ := numFloat(hero, "CBodyComponent.m_vecY")
				if okY {
					iv.X = float64(cx)*128.0 + vx
					iv.Y = float64(cy)*128.0 + vy
					iv.HasPos = true
				}
			}
		}
		if si.heroHandle != 0 && si.heroHandle != 0xFFFFFF {
			if unit := st.heroName(si.heroHandle); unit != "" {
				st.slots[slot].hero = unit
				st.ag.setSlot(slot, si.team, unit)
			}
		}
		iv.Kills, _ = numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iKills", si.prIndex))
		iv.Deaths, _ = numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iDeaths", si.prIndex))
		iv.Assists, _ = numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iAssists", si.prIndex))
		iv.Level, _ = numInt(st.pr, fmt.Sprintf("m_vecPlayerTeamData.%04d.m_iLevel", si.prIndex))

		td := st.teams[si.team]
		if td != nil && si.teamSlot >= 0 {
			base := fmt.Sprintf("m_vecDataTeam.%04d.", si.teamSlot)
			iv.Gold, _ = numInt(td, base+"m_iTotalEarnedGold")
			iv.XP, _ = numInt(td, base+"m_iTotalEarnedXP")
			iv.LastHits, _ = numInt(td, base+"m_iLastHitCount")
			iv.Networth, _ = numInt(td, base+"m_iNetWorth")
			iv.CampsStacked, _ = numInt(td, base+"m_iCampsStacked")
			iv.CreepsStacked, _ = numInt(td, base+"m_iCreepsStacked")
			iv.RunePickups, _ = numInt(td, base+"m_iRunePickups")
			iv.WisdomsCaptured, _ = numInt(td, base+"m_iWisdomShrinesTaken")
			iv.WatchersCaptured, _ = numInt(td, base+"m_iWatchersTaken")
			iv.LotusesGathered, _ = numInt(td, base+"m_iLotusesTaken")
			iv.SentryWards, _ = numInt(td, base+"m_iSentryWardsPlaced")
			if v, ok := numFloat(td, base+"m_fStuns"); ok {
				iv.Stuns = v
			}
		}
		ivs = append(ivs, iv)
	}
	st.scanStartingItems(raw)
	if !st.startKnown {
		st.pendingIntervals = append(st.pendingIntervals, ivs...)
		return
	}
	for _, iv := range ivs {
		iv.Time -= st.gameStart
		st.ag.onInterval(iv)
	}
	st.emitAbilityLevels(raw - st.gameStart)
}

func (st *replayState) flushIntervals() {
	for _, iv := range st.pendingIntervals {
		iv.Time -= st.gameStart
		st.ag.onInterval(iv)
	}
	st.pendingIntervals = nil
}

func (st *replayState) scanStartingItems(t int) {
	for slot := 0; slot < 10; slot++ {
		if st.startingItemsWritten[slot] {
			continue
		}
		si := st.slots[slot]
		hero := st.p.FindEntityByHandle(si.heroHandle)
		if hero == nil {
			continue
		}
		found := false
		heroName := unitToHeroName(hero.GetClassName())
		for i := 0; i < 9; i++ {
			h, ok := numUint64(hero, fmt.Sprintf("m_hItems.%04d", i))
			if !ok || h == 0 || h == 0xFFFFFF {
				continue
			}
			found = true
			item := st.entityName(h)
			if _, ok := ItemWardCosts[item]; ok {
				st.ag.onPurchase(combatEvent{Time: t, ValueName: item, TargetName: heroName})
			}
		}
		if found {
			st.startingItemsWritten[slot] = true
		}
	}
}

func (st *replayState) emitAbilityLevels(t int) {
	for slot := 0; slot < 10; slot++ {
		si := st.slots[slot]
		hero := st.p.FindEntityByHandle(si.heroHandle)
		if hero == nil {
			continue
		}
		heroName := unitToHeroName(hero.GetClassName())
		for i := 0; i < 32; i++ {
			h, ok := numUint64(hero, fmt.Sprintf("m_vecAbilities.%04d", i))
			if !ok || h == 0 || h == 0xFFFFFF {
				h, ok = numUint64(hero, fmt.Sprintf("m_hAbilities.%04d", i))
				if !ok || h == 0 || h == 0xFFFFFF {
					continue
				}
			}
			name, ok := st.abilityNames[h]
			if !ok {
				name = st.entityName(h)
				st.abilityNames[h] = name
			}
			if !PurgeTalents[name] {
				continue
			}
			ae := st.p.FindEntityByHandle(h)
			if ae == nil {
				continue
			}
			lvl, _ := numInt(ae, "m_iLevel")
			st.ag.onAbilityLevel(heroName, name, lvl, t)
		}
	}
}

func (st *replayState) entityName(handle uint64) string {
	e := st.p.FindEntityByHandle(handle)
	if e == nil {
		return ""
	}
	idx, ok := numInt(e, "m_pEntity.m_nameStringTableIndex")
	if !ok {
		idx, _ = numInt(e, "m_pEntity.m_nameStringableIndex")
	}
	if name, ok := st.p.LookupStringByIndex("EntityNames", int32(idx)); ok {
		return name
	}
	return ""
}

func (st *replayState) finishRoster() {
	if !st.startKnown {
		st.gameStart = 0
		st.startKnown = true
		st.flushIntervals()
		st.flushPending()
	}
	if len(st.ag.roster) == 0 {
		players := make([]playerInfo, 0, 10)
		for slot := 0; slot < 10; slot++ {
			si := st.slots[slot]
			hero := st.slots[slot].hero
			if hero == "" {
				hero = st.heroName(si.heroHandle)
			}
			players = append(players, playerInfo{
				Slot:    slot,
				SteamID: si.steamID,
				Hero:    hero,
				Name:    si.name,
				Team:    si.team,
			})
		}
		dur := float64(st.lastCombatTime - st.gameStart)
		if dur < 0 {
			dur = 0
		}
		st.ag.setEpilogue(st.matchID, shortestFloat32(dur), st.lastTBKillTeam == teamDire, players)
	}
}

func playerIDOf(e *manta.Entity) int32 {
	if e == nil {
		return 0
	}
	if v, ok := e.GetInt32("m_iPlayerID"); ok {
		return v
	}
	if v, ok := e.GetUint32("m_iPlayerID"); ok {
		return int32(v)
	}
	return 0
}

func int32OrUint32(e *manta.Entity, name string) (int32, bool) {
	if e == nil {
		return 0, false
	}
	if v, ok := e.GetInt32(name); ok {
		return v, true
	}
	if v, ok := e.GetUint32(name); ok {
		return int32(v), true
	}
	return 0, false
}

func numInt(e *manta.Entity, name string) (int, bool) {
	if e == nil {
		return 0, false
	}
	switch v := e.Get(name).(type) {
	case int32:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case int64:
		return int(v), true
	}
	return 0, false
}

func numFloat(e *manta.Entity, name string) (float64, bool) {
	if e == nil {
		return 0, false
	}
	switch v := e.Get(name).(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

func numUint64(e *manta.Entity, name string) (uint64, bool) {
	if e == nil {
		return 0, false
	}
	if v, ok := e.GetUint64(name); ok {
		return v, true
	}
	if v, ok := e.GetUint32(name); ok {
		return uint64(v), true
	}
	if v, ok := e.GetInt32(name); ok {
		return uint64(uint32(v)), true
	}
	return 0, false
}
