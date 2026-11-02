package stats

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// playerInfo is one roster entry from CDemoFileInfo (or the PlayerResource
// fallback). Slot is the 0..9 index the parser assigns, not a raw PR index.
type playerInfo struct {
	Slot    int
	SteamID int64
	Hero    string
	Name    string
	Team    string
}

// intervalData is one per-second snapshot of a player.
type intervalData struct {
	Slot             int
	Time             int
	Unit             string
	LifeState        int
	HasLifeState     bool
	Gold             int
	TowerDamage      int
	LastHits         int
	XP               int
	X, Y             float64
	HasPos           bool
	Stuns            float64
	HeroID           int
	Level            int
	Kills            int
	Deaths           int
	Assists          int
	CampsStacked     int
	CreepsStacked    int
	RunePickups      int
	WisdomsCaptured  int
	WatchersCaptured int
	LotusesGathered  int
	SentryWards      int
	Networth         int
}

// combatEvent is a resolved combat-log entry.
type combatEvent struct {
	Time             int
	Type             string
	AttackerName     string
	TargetName       string
	SourceName       string
	TargetSourceName string
	Inflictor        string
	ValueName        string
	AttackerHero     bool
	TargetHero       bool
	AttackerIllusion bool
	TargetIllusion   bool
	Value            int
	StunDuration     float64
	GoldReason       int
	AbilityLevel     int
}

type srcKey struct {
	Attacker  string
	Inflictor string
}

type buffKey struct {
	Attacker  string
	Category  string
	Inflictor string
}

type modKey struct {
	Attacker  string
	Target    string
	Inflictor string
}

type activeMod struct {
	Inflictor string
	Category  string
	AddTime   int
}

type healAgg struct {
	Duration float64
	Value    float64
}

type healActive struct {
	Attacker string
	Clean    string
	AddTime  int
}

type talentKey struct {
	Hero   string
	Talent string
}

type aggregator struct {
	m *Result

	roster   []playerInfo
	slotHero map[int]string
	slotTeam map[int]string
	heroTeam map[string]string

	lastInterval     map[int]intervalData
	lastIntervalTime int

	deadSecondsBySlot map[int]float64
	lastLifeState     map[int]int
	stateTime         map[int]int

	roleSamples map[int]*roleSamples
	roleFarm    map[int]roleFarm

	healingByAttacker     map[string]int
	heroDamageByAttacker  map[string]int
	damageTakenByTarget   map[string]int
	towerDamageByAttacker map[string]int

	stunSources   map[srcKey]float64
	buffSources   map[buffKey]float64
	healSources   map[srcKey]healAgg
	purgeCounts   map[srcKey]int
	healsByTarget map[string][]healActive
	shardAt       map[string]int
	talents       map[talentKey]int
	durations     [kindCount]map[srcKey]float64
	active        [kindCount]map[modKey][]activeMod

	goldSpentSmoke map[string]int
	goldSpentDust  map[string]int
	goldLostByHero map[string]int
	courierKills   map[string]int

	firstBloodSlot int

	mods map[string]modClass
}

func newAggregator() *aggregator {
	a := &aggregator{
		m:                     &Result{},
		slotHero:              map[int]string{},
		slotTeam:              map[int]string{},
		heroTeam:              map[string]string{},
		lastInterval:          map[int]intervalData{},
		deadSecondsBySlot:     map[int]float64{},
		lastLifeState:         map[int]int{},
		stateTime:             map[int]int{},
		roleSamples:           map[int]*roleSamples{},
		roleFarm:              map[int]roleFarm{},
		healingByAttacker:     map[string]int{},
		heroDamageByAttacker:  map[string]int{},
		damageTakenByTarget:   map[string]int{},
		towerDamageByAttacker: map[string]int{},
		stunSources:           map[srcKey]float64{},
		buffSources:           map[buffKey]float64{},
		healSources:           map[srcKey]healAgg{},
		purgeCounts:           map[srcKey]int{},
		healsByTarget:         map[string][]healActive{},
		shardAt:               map[string]int{},
		talents:               map[talentKey]int{},
		goldSpentSmoke:        map[string]int{},
		goldSpentDust:         map[string]int{},
		goldLostByHero:        map[string]int{},
		courierKills:          map[string]int{},
		firstBloodSlot:        -1,
		mods:                  map[string]modClass{},
	}
	return a
}

func (a *aggregator) setEpilogue(matchID int64, duration float64, radiantWin bool, players []playerInfo) {
	a.m.MatchID = matchID
	a.m.DurationSec = duration
	a.m.RadiantWin = radiantWin
	a.roster = players
}

func (a *aggregator) setSlot(slot int, team, hero string) {
	a.slotTeam[slot] = team
	if hero != "" {
		a.slotHero[slot] = hero
		a.heroTeam[heroKey(hero)] = team
	}
}

func (a *aggregator) classify(inf string) modClass {
	if c, ok := a.mods[inf]; ok {
		return c
	}
	c := classifyModifier(inf)
	a.mods[inf] = c
	return c
}

func (a *aggregator) gateOK(gate purgeGate, talent, attacker string, t int) bool {
	switch gate {
	case gateShard:
		at, ok := a.shardAt[heroKey(attacker)]
		return ok && t >= at
	case gateTalent:
		at, ok := a.talents[talentKey{heroKey(attacker), talent}]
		return ok && t >= at
	}
	return true
}

func (a *aggregator) heroIsAlly(x, y string) bool {
	tx, okX := a.heroTeam[heroKey(x)]
	ty, okY := a.heroTeam[heroKey(y)]
	return okX && okY && tx == ty
}

func (a *aggregator) onInterval(iv intervalData) {
	a.lastInterval[iv.Slot] = iv
	if iv.Time > a.lastIntervalTime {
		a.lastIntervalTime = iv.Time
	}
	if iv.HasLifeState {
		if prev, ok := a.lastLifeState[iv.Slot]; ok && prev != 0 {
			a.deadSecondsBySlot[iv.Slot] += float64(iv.Time - a.stateTime[iv.Slot])
		}
		a.lastLifeState[iv.Slot] = iv.LifeState
		a.stateTime[iv.Slot] = iv.Time
	}
	if iv.Time >= 0 && iv.Time <= roleWindowSec {
		a.roleFarm[iv.Slot] = roleFarm{networth: iv.Networth, lh: iv.LastHits, xp: iv.XP}
		if iv.HasPos && (!iv.HasLifeState || iv.LifeState == 0) {
			rs := a.roleSamples[iv.Slot]
			if rs == nil {
				rs = &roleSamples{}
				a.roleSamples[iv.Slot] = rs
			}
			rs.sample(iv.X, iv.Y)
		}
	}
}

func (a *aggregator) onDamage(ev combatEvent) {
	key := ev.AttackerName
	if strings.HasPrefix(ev.SourceName, "npc_dota_hero_") {
		key = ev.SourceName
	}
	attackerHero := ev.AttackerHero && !ev.AttackerIllusion
	targetHero := ev.TargetHero && !ev.TargetIllusion
	if targetHero && key != ev.TargetName {
		a.heroDamageByAttacker[heroKey(key)] += ev.Value
	}
	if attackerHero && isBuilding(ev.TargetName) {
		a.towerDamageByAttacker[heroKey(key)] += ev.Value
	}
	if targetHero && attackerHero && key != ev.TargetName {
		a.damageTakenByTarget[heroKey(ev.TargetName)] += ev.Value
	}
}

func (a *aggregator) onHeal(ev combatEvent) {
	if !ev.AttackerHero || !ev.TargetHero {
		return
	}
	attacker, target := ev.AttackerName, ev.TargetName
	if attacker == "" || target == "" {
		return
	}
	if attacker == target {
		a.healTick(target, ev)
		return
	}
	if !a.heroIsAlly(attacker, target) {
		return
	}
	a.healingByAttacker[heroKey(attacker)] += ev.Value
	if InstantHealItems[ev.Inflictor] {
		sk := srcKey{heroKey(attacker), ev.Inflictor}
		agg := a.healSources[sk]
		agg.Value += float64(ev.Value)
		a.healSources[sk] = agg
	}
}

func (a *aggregator) healTick(target string, ev combatEvent) {
	if ev.Inflictor != "" && ev.Inflictor != "dota_unknown" {
		return
	}
	var best healActive
	found := false
	for _, h := range a.healsByTarget[target] {
		if h.Attacker == target {
			continue
		}
		if !found || h.AddTime >= best.AddTime {
			best, found = h, true
		}
	}
	if !found || ev.Value <= 0 {
		return
	}
	hk := heroKey(best.Attacker)
	a.healingByAttacker[hk] += ev.Value
	sk := srcKey{hk, best.Clean}
	agg := a.healSources[sk]
	agg.Value += float64(ev.Value)
	a.healSources[sk] = agg
}

func (a *aggregator) removeHealActive(target, attacker, clean string) {
	list := a.healsByTarget[target]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Attacker == attacker && list[i].Clean == clean {
			a.healsByTarget[target] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

func (a *aggregator) push(kind modKind, attacker, target, clean, category string, t int) {
	k := modKey{attacker, target, clean}
	m := a.active[kind]
	if m == nil {
		m = map[modKey][]activeMod{}
		a.active[kind] = m
	}
	m[k] = append(m[k], activeMod{Inflictor: clean, Category: category, AddTime: t})
}

func (a *aggregator) popDuration(kind modKind, attacker, target, clean string, t int) {
	k := modKey{attacker, target, clean}
	list := a.active[kind][k]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Inflictor != clean {
			continue
		}
		if d := float64(t - list[i].AddTime); d > 0 {
			m := a.durations[kind]
			if m == nil {
				m = map[srcKey]float64{}
				a.durations[kind] = m
			}
			m[srcKey{heroKey(attacker), clean}] += d
		}
		a.active[kind][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *aggregator) popHeal(attacker, target, clean string, t int) {
	k := modKey{attacker, target, clean}
	list := a.active[kindHeal][k]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Inflictor != clean {
			continue
		}
		if d := float64(t - list[i].AddTime); d > 0 {
			sk := srcKey{heroKey(attacker), clean}
			agg := a.healSources[sk]
			agg.Duration += d
			a.healSources[sk] = agg
		}
		a.active[kindHeal][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *aggregator) popBuff(attacker, target, clean string, t int) {
	k := modKey{attacker, target, clean}
	list := a.active[kindBuff][k]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Inflictor != clean {
			continue
		}
		if d := float64(t - list[i].AddTime); d > 0 {
			a.buffSources[buffKey{heroKey(attacker), list[i].Category, clean}] += d
		}
		a.active[kindBuff][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *aggregator) onModifier(ev combatEvent, add bool) {
	if !ev.TargetHero || ev.TargetIllusion {
		return
	}
	inf := ev.Inflictor
	if inf == "" {
		return
	}
	attacker := ev.AttackerName
	if attacker == "" {
		attacker = ev.TargetName
	}
	target := ev.TargetName
	if attacker == target {
		return
	}
	c := a.classify(inf)
	if add {
		if c.purge && a.heroIsAlly(attacker, target) && a.gateOK(c.gate, c.talent, attacker, ev.Time) {
			a.purgeCounts[srcKey{heroKey(attacker), c.clean}]++
		}
		switch c.kind {
		case kindBreak, kindFear, kindRoot, kindLeash, kindTrap, kindTaunt, kindSilence, kindDisarm:
			if !a.heroIsAlly(attacker, target) {
				a.push(c.kind, attacker, target, c.clean, "", ev.Time)
			}
		case kindHeal:
			a.push(kindHeal, attacker, target, c.clean, "", ev.Time)
			a.healsByTarget[target] = append(a.healsByTarget[target], healActive{Attacker: attacker, Clean: c.clean, AddTime: ev.Time})
		case kindBuff:
			if !trackedBuff[c.category] || !a.heroIsAlly(attacker, target) {
				return
			}
			a.push(kindBuff, attacker, target, c.clean, c.category, ev.Time)
			if ev.StunDuration > 0 {
				a.stunSources[srcKey{heroKey(attacker), c.clean}] += ev.StunDuration
			}
		}
		return
	}
	switch c.kind {
	case kindBreak, kindFear, kindRoot, kindLeash, kindTrap, kindTaunt, kindSilence, kindDisarm:
		if !a.heroIsAlly(attacker, target) {
			a.popDuration(c.kind, attacker, target, c.clean, ev.Time)
		}
	case kindHeal:
		a.popHeal(attacker, target, c.clean, ev.Time)
		a.removeHealActive(target, attacker, c.clean)
	case kindBuff:
		if a.heroIsAlly(attacker, target) {
			a.popBuff(attacker, target, c.clean, ev.Time)
		}
	}
}

func (a *aggregator) onPurchase(ev combatEvent) {
	if ev.ValueName == "" {
		return
	}
	if ev.ValueName == "item_aghanims_shard" && ev.TargetName != "" {
		hk := heroKey(ev.TargetName)
		if _, ok := a.shardAt[hk]; !ok {
			a.shardAt[hk] = ev.Time
		}
		return
	}
	cost, ok := ItemWardCosts[ev.ValueName]
	if !ok {
		return
	}
	switch {
	case strings.Contains(ev.ValueName, "smoke"):
		a.goldSpentSmoke[heroKey(ev.TargetName)] += cost
	case strings.Contains(ev.ValueName, "dust"):
		a.goldSpentDust[heroKey(ev.TargetName)] += cost
	}
}

func (a *aggregator) onGold(ev combatEvent) {
	if ev.GoldReason == 1 && ev.TargetName != "" {
		v := ev.Value
		if v < 0 {
			v = -v
		}
		a.goldLostByHero[heroKey(ev.TargetName)] += v
	}
}

func (a *aggregator) onDeath(ev combatEvent) {
	if ev.TargetName == "npc_dota_courier" && ev.SourceName != "" {
		a.courierKills[heroKey(ev.SourceName)]++
	}
}

func (a *aggregator) onAbility(ev combatEvent) {
	inf := ev.Inflictor
	if inf == "" || !ev.AttackerHero || ev.AttackerName == "" {
		return
	}
	hk := heroKey(ev.AttackerName)
	if _, ok := a.heroTeam[hk]; !ok {
		return
	}
	if talent, ok := TalentGlobalAbilities[inf]; ok {
		if a.gateOK(gateTalent, talent, ev.AttackerName, ev.Time) {
			a.purgeCounts[srcKey{hk, canonicalName(inf)}]++
		}
		return
	}
	target := ev.TargetName
	if talent, ok := TalentPurgeAbilities[inf]; ok &&
		ev.TargetHero && !ev.TargetIllusion &&
		target != "" && ev.AttackerName != target && a.heroIsAlly(ev.AttackerName, target) &&
		a.gateOK(gateTalent, talent, ev.AttackerName, ev.Time) {
		a.purgeCounts[srcKey{hk, canonicalName(inf)}]++
	}
}

func (a *aggregator) onAbilityLevel(hero, ability string, level, t int) {
	if level <= 0 || hero == "" || !PurgeTalents[ability] {
		return
	}
	k := talentKey{heroKey(hero), ability}
	if _, ok := a.talents[k]; !ok {
		a.talents[k] = t
	}
}

func (a *aggregator) onMiniboss(slot, t int) {
	hero, ok := a.slotHero[slot]
	if !ok {
		return
	}
	hk := heroKey(hero)
	if at, ok := a.shardAt[hk]; !ok || t < at {
		a.shardAt[hk] = t
	}
}

func (a *aggregator) onFirstBlood(slot int) {
	a.firstBloodSlot = slot
}

func (a *aggregator) result() *Result {
	players := append([]playerInfo(nil), a.roster...)
	if len(players) == 0 {
		for slot, hero := range a.slotHero {
			players = append(players, playerInfo{Slot: slot, Hero: hero, Team: a.slotTeam[slot]})
		}
	}
	for i := range players {
		if hero, ok := a.slotHero[players[i].Slot]; ok {
			players[i].Hero = hero
		}
		if team, ok := a.slotTeam[players[i].Slot]; ok && players[i].Team == "" {
			players[i].Team = team
		}
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Slot < players[j].Slot })

	a.m.Players = make([]Player, len(players))
	heroKeys := make([]string, len(players))
	for i := range players {
		p := &players[i]
		mp := &a.m.Players[i]
		mp.SteamID = p.SteamID
		mp.PlayerID = p.Slot
		mp.Hero = strings.TrimPrefix(p.Hero, "npc_dota_hero_")
		mp.HeroID = heroID(p.Hero)
		mp.Team = p.Team
		mp.Name = p.Name
		if iv, ok := a.lastInterval[p.Slot]; ok {
			mp.Kills = iv.Kills
			mp.Deaths = iv.Deaths
			mp.Assists = iv.Assists
			mp.Level = iv.Level
			mp.LastHits = iv.LastHits
			mp.Networth = iv.Networth
			mp.StunDuration = shortestFloat32(iv.Stuns)
			mp.CampsStacked = iv.CampsStacked
			mp.CreepsStacked = iv.CreepsStacked
			mp.RunePickups = iv.RunePickups
			mp.WisdomsCaptured = iv.WisdomsCaptured
			mp.WatchersCaptured = iv.WatchersCaptured
			mp.LotusesGathered = iv.LotusesGathered
			mp.FirstBlood = a.firstBloodSlot == p.Slot
			mp.GoldSpentWards = iv.SentryWards * 50
			if a.lastIntervalTime > 0 {
				minutes := float64(a.lastIntervalTime) / 60.0
				mp.GPM = int(math.Round(float64(iv.Gold) / minutes))
				mp.XPM = int(math.Round(float64(iv.XP) / minutes))
			}
		}
		hk := heroKey(p.Hero)
		heroKeys[i] = hk
		mp.Healing = a.healingByAttacker[hk]
		mp.HeroDamage = a.heroDamageByAttacker[hk]
		mp.DamageTaken = a.damageTakenByTarget[hk]
		mp.TowerDamage = a.towerDamageByAttacker[hk]
		mp.GoldSpentSmoke = a.goldSpentSmoke[hk]
		mp.GoldSpentDust = a.goldSpentDust[hk]
		mp.GoldLost = a.goldLostByHero[hk]
		mp.CourierKills = a.courierKills[hk]
		mp.TimeDead = math.Round(a.deadSecondsBySlot[p.Slot])
	}

	byHero := make(map[string][]*Player, len(a.m.Players))
	for i := range a.m.Players {
		byHero[heroKeys[i]] = append(byHero[heroKeys[i]], &a.m.Players[i])
	}

	for k, v := range a.stunSources {
		for _, mp := range byHero[k.Attacker] {
			mp.StunSources = append(mp.StunSources, SourceEntry{
				Inflictor: k.Inflictor,
				Category:  "stun",
				Duration:  v,
			})
		}
	}
	for k, v := range a.buffSources {
		for _, mp := range byHero[k.Attacker] {
			mp.BuffSources = append(mp.BuffSources, SourceEntry{
				Inflictor: k.Inflictor,
				Category:  k.Category,
				Duration:  math.Round(v),
			})
			mp.BuffDuration += v
		}
	}
	for k, n := range a.purgeCounts {
		for _, mp := range byHero[k.Attacker] {
			mp.BuffSources = append(mp.BuffSources, SourceEntry{
				Inflictor: k.Inflictor,
				Category:  "purge",
				Count:     n,
			})
		}
	}

	specs := []struct {
		kind     modKind
		category string
		entries  func(*Player) *[]SourceEntry
		total    func(*Player) *float64
	}{
		{kindBreak, "break", func(p *Player) *[]SourceEntry { return &p.BreakSources }, func(p *Player) *float64 { return &p.BreakDuration }},
		{kindFear, "fear", func(p *Player) *[]SourceEntry { return &p.FearSources }, func(p *Player) *float64 { return &p.FearDuration }},
		{kindRoot, "root", func(p *Player) *[]SourceEntry { return &p.RootSources }, func(p *Player) *float64 { return &p.RootsDuration }},
		{kindLeash, "leash", func(p *Player) *[]SourceEntry { return &p.LeashSources }, func(p *Player) *float64 { return &p.LeashDuration }},
		{kindTrap, "trap", func(p *Player) *[]SourceEntry { return &p.TrapSources }, func(p *Player) *float64 { return &p.TrapDuration }},
		{kindTaunt, "taunt", func(p *Player) *[]SourceEntry { return &p.TauntSources }, func(p *Player) *float64 { return &p.TauntDuration }},
		{kindSilence, "silence", func(p *Player) *[]SourceEntry { return &p.SilenceSources }, func(p *Player) *float64 { return &p.SilenceDuration }},
		{kindDisarm, "disarm", func(p *Player) *[]SourceEntry { return &p.DisarmSources }, func(p *Player) *float64 { return &p.DisarmDuration }},
	}
	for _, s := range specs {
		for k, v := range a.durations[s.kind] {
			for _, mp := range byHero[k.Attacker] {
				*s.entries(mp) = append(*s.entries(mp), SourceEntry{
					Inflictor: k.Inflictor,
					Category:  s.category,
					Duration:  math.Round(v),
				})
				*s.total(mp) += v
			}
		}
	}
	for k, agg := range a.healSources {
		for _, mp := range byHero[k.Attacker] {
			entry := SourceEntry{Inflictor: k.Inflictor, Category: "heal"}
			if agg.Duration > 0 {
				entry.Duration = math.Round(agg.Duration)
				mp.HealDuration += agg.Duration
			}
			if agg.Value > 0 {
				entry.Value = math.Round(agg.Value)
				mp.HealValue += agg.Value
			}
			mp.HealSources = append(mp.HealSources, entry)
		}
	}

	for i := range a.m.Players {
		mp := &a.m.Players[i]
		sortSources(mp.StunSources)
		sortSources(mp.BuffSources)
		sortHealSources(mp.HealSources)
		for _, s := range specs {
			sortSources(*s.entries(mp))
			*s.total(mp) = math.Round(*s.total(mp))
		}
		mp.BuffDuration = math.Round(mp.BuffDuration)
		mp.HealDuration = math.Round(mp.HealDuration)
		mp.HealValue = math.Round(mp.HealValue)
	}

	roleInputs := make([]roleInput, 0, len(a.m.Players))
	for i := range a.m.Players {
		slot := players[i].Slot
		farm, ok := a.roleFarm[slot]
		if !ok || (farm.networth == 0 && farm.lh == 0 && farm.xp == 0) {
			if iv, ok := a.lastInterval[slot]; ok {
				farm = roleFarm{networth: iv.Networth, lh: iv.LastHits, xp: iv.XP}
			}
		}
		roleInputs = append(roleInputs, roleInput{
			p:     &a.m.Players[i],
			stats: a.roleSamples[slot],
			farm:  farm,
		})
	}
	assignRoles(roleInputs)
	return a.m
}

func sortSources(s []SourceEntry) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Duration != s[j].Duration {
			return s[i].Duration > s[j].Duration
		}
		if s[i].Inflictor != s[j].Inflictor {
			return s[i].Inflictor < s[j].Inflictor
		}
		return s[i].Category < s[j].Category
	})
}

func sortHealSources(s []SourceEntry) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Duration != s[j].Duration {
			return s[i].Duration > s[j].Duration
		}
		if s[i].Value != s[j].Value {
			return s[i].Value > s[j].Value
		}
		return s[i].Inflictor < s[j].Inflictor
	})
}

func shortestFloat32(v float64) float64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', -1, 32), 64)
	return f
}
