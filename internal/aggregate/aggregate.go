package aggregate

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/drawiks/manta-cli/internal/catalog"
	"github.com/drawiks/manta-cli/internal/events"
	"github.com/drawiks/manta-cli/internal/roles"
	"github.com/drawiks/manta-cli/stats"
)

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

type Aggregator struct {
	m *stats.Match

	roster   []events.PlayerInfo
	slotHero map[int]string
	slotTeam map[int]string
	heroTeam map[string]string

	lastInterval     map[int]events.Interval
	lastIntervalTime int

	deadSecondsBySlot map[int]float64
	lastLifeState     map[int]int
	stateTime         map[int]int

	roleSamples map[int]*roles.Samples
	roleFarm    map[int]roles.Farm

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
	durations     [catalog.KindCount]map[srcKey]float64
	active        [catalog.KindCount]map[modKey][]activeMod

	goldSpentSmoke map[string]int
	goldSpentDust  map[string]int
	goldLostByHero map[string]int
	courierKills   map[string]int

	firstBloodSlot int

	mods map[string]catalog.ModClass
}

func New() *Aggregator {
	a := &Aggregator{
		m:                     &stats.Match{},
		slotHero:              map[int]string{},
		slotTeam:              map[int]string{},
		heroTeam:              map[string]string{},
		lastInterval:          map[int]events.Interval{},
		deadSecondsBySlot:     map[int]float64{},
		lastLifeState:         map[int]int{},
		stateTime:             map[int]int{},
		roleSamples:           map[int]*roles.Samples{},
		roleFarm:              map[int]roles.Farm{},
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
		mods:                  map[string]catalog.ModClass{},
	}
	return a
}

func (a *Aggregator) Epilogue(e events.Epilogue) {
	a.m.MatchID = e.MatchID
	a.m.DurationSec = e.DurationSec
	a.m.RadiantWin = e.RadiantWin
	a.roster = e.Players
}

func (a *Aggregator) Slot(e events.Slot) {
	a.slotTeam[e.Slot] = e.Team
	if e.Hero != "" {
		a.slotHero[e.Slot] = e.Hero
		a.heroTeam[catalog.HeroKey(e.Hero)] = e.Team
	}
}

func (a *Aggregator) classify(inf string) catalog.ModClass {
	if c, ok := a.mods[inf]; ok {
		return c
	}
	c := catalog.ClassifyModifier(inf)
	a.mods[inf] = c
	return c
}

func (a *Aggregator) gateOK(gate catalog.PurgeGate, talent, attacker string, t int) bool {
	switch gate {
	case catalog.GateShard:
		at, ok := a.shardAt[catalog.HeroKey(attacker)]
		return ok && t >= at
	case catalog.GateTalent:
		at, ok := a.talents[talentKey{catalog.HeroKey(attacker), talent}]
		return ok && t >= at
	}
	return true
}

func (a *Aggregator) heroIsAlly(x, y string) bool {
	tx, okX := a.heroTeam[catalog.HeroKey(x)]
	ty, okY := a.heroTeam[catalog.HeroKey(y)]
	return okX && okY && tx == ty
}

func (a *Aggregator) Interval(iv events.Interval) {
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
	if iv.Time >= 0 && iv.Time <= roles.WindowSec {
		a.roleFarm[iv.Slot] = roles.Farm{Networth: iv.Networth, LH: iv.LastHits, XP: iv.XP}
		if iv.HasPos && (!iv.HasLifeState || iv.LifeState == 0) {
			rs := a.roleSamples[iv.Slot]
			if rs == nil {
				rs = &roles.Samples{}
				a.roleSamples[iv.Slot] = rs
			}
			rs.Sample(iv.X, iv.Y)
		}
	}
}

func (a *Aggregator) onDamage(ev events.Combat) {
	key := ev.AttackerName
	if strings.HasPrefix(ev.SourceName, "npc_dota_hero_") {
		key = ev.SourceName
	}
	attackerHero := ev.AttackerHero && !ev.AttackerIllusion
	targetHero := ev.TargetHero && !ev.TargetIllusion
	if targetHero && key != ev.TargetName {
		a.heroDamageByAttacker[catalog.HeroKey(key)] += ev.Value
	}
	if attackerHero && catalog.IsBuilding(ev.TargetName) {
		a.towerDamageByAttacker[catalog.HeroKey(key)] += ev.Value
	}
	if targetHero && attackerHero && key != ev.TargetName {
		a.damageTakenByTarget[catalog.HeroKey(ev.TargetName)] += ev.Value
	}
}

func (a *Aggregator) onHeal(ev events.Combat) {
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
	a.healingByAttacker[catalog.HeroKey(attacker)] += ev.Value
	if catalog.InstantHealItems[ev.Inflictor] {
		sk := srcKey{catalog.HeroKey(attacker), ev.Inflictor}
		agg := a.healSources[sk]
		agg.Value += float64(ev.Value)
		a.healSources[sk] = agg
	}
}

func (a *Aggregator) healTick(target string, ev events.Combat) {
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
	hk := catalog.HeroKey(best.Attacker)
	a.healingByAttacker[hk] += ev.Value
	sk := srcKey{hk, best.Clean}
	agg := a.healSources[sk]
	agg.Value += float64(ev.Value)
	a.healSources[sk] = agg
}

func (a *Aggregator) removeHealActive(target, attacker, clean string) {
	list := a.healsByTarget[target]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Attacker == attacker && list[i].Clean == clean {
			a.healsByTarget[target] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

func (a *Aggregator) push(kind catalog.ModKind, attacker, target, clean, category string, t int) {
	k := modKey{attacker, target, clean}
	m := a.active[kind]
	if m == nil {
		m = map[modKey][]activeMod{}
		a.active[kind] = m
	}
	m[k] = append(m[k], activeMod{Inflictor: clean, Category: category, AddTime: t})
}

func (a *Aggregator) popDuration(kind catalog.ModKind, attacker, target, clean string, t int) {
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
			m[srcKey{catalog.HeroKey(attacker), clean}] += d
		}
		a.active[kind][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *Aggregator) popHeal(attacker, target, clean string, t int) {
	k := modKey{attacker, target, clean}
	list := a.active[catalog.KindHeal][k]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Inflictor != clean {
			continue
		}
		if d := float64(t - list[i].AddTime); d > 0 {
			sk := srcKey{catalog.HeroKey(attacker), clean}
			agg := a.healSources[sk]
			agg.Duration += d
			a.healSources[sk] = agg
		}
		a.active[catalog.KindHeal][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *Aggregator) popBuff(attacker, target, clean string, t int) {
	k := modKey{attacker, target, clean}
	list := a.active[catalog.KindBuff][k]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Inflictor != clean {
			continue
		}
		if d := float64(t - list[i].AddTime); d > 0 {
			a.buffSources[buffKey{catalog.HeroKey(attacker), list[i].Category, clean}] += d
		}
		a.active[catalog.KindBuff][k] = append(list[:i], list[i+1:]...)
		return
	}
}

func (a *Aggregator) onModifier(ev events.Combat, add bool) {
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
		if c.Purge && a.heroIsAlly(attacker, target) && a.gateOK(c.Gate, c.Talent, attacker, ev.Time) {
			a.purgeCounts[srcKey{catalog.HeroKey(attacker), c.Clean}]++
		}
		switch c.Kind {
		case catalog.KindBreak, catalog.KindFear, catalog.KindRoot, catalog.KindLeash, catalog.KindTrap, catalog.KindTaunt, catalog.KindSilence, catalog.KindDisarm:
			if !a.heroIsAlly(attacker, target) {
				a.push(c.Kind, attacker, target, c.Clean, "", ev.Time)
			}
		case catalog.KindHeal:
			a.push(catalog.KindHeal, attacker, target, c.Clean, "", ev.Time)
			a.healsByTarget[target] = append(a.healsByTarget[target], healActive{Attacker: attacker, Clean: c.Clean, AddTime: ev.Time})
		case catalog.KindBuff:
			if !catalog.TrackedBuff[c.Category] || !a.heroIsAlly(attacker, target) {
				return
			}
			a.push(catalog.KindBuff, attacker, target, c.Clean, c.Category, ev.Time)
			if ev.StunDuration > 0 {
				a.stunSources[srcKey{catalog.HeroKey(attacker), c.Clean}] += ev.StunDuration
			}
		}
		return
	}
	switch c.Kind {
	case catalog.KindBreak, catalog.KindFear, catalog.KindRoot, catalog.KindLeash, catalog.KindTrap, catalog.KindTaunt, catalog.KindSilence, catalog.KindDisarm:
		if !a.heroIsAlly(attacker, target) {
			a.popDuration(c.Kind, attacker, target, c.Clean, ev.Time)
		}
	case catalog.KindHeal:
		a.popHeal(attacker, target, c.Clean, ev.Time)
		a.removeHealActive(target, attacker, c.Clean)
	case catalog.KindBuff:
		if a.heroIsAlly(attacker, target) {
			a.popBuff(attacker, target, c.Clean, ev.Time)
		}
	}
}

func (a *Aggregator) onPurchase(ev events.Combat) {
	if ev.ValueName == "" {
		return
	}
	if ev.ValueName == "item_aghanims_shard" && ev.TargetName != "" {
		hk := catalog.HeroKey(ev.TargetName)
		if _, ok := a.shardAt[hk]; !ok {
			a.shardAt[hk] = ev.Time
		}
		return
	}
	cost, ok := catalog.ItemWardCosts[ev.ValueName]
	if !ok {
		return
	}
	switch {
	case strings.Contains(ev.ValueName, "smoke"):
		a.goldSpentSmoke[catalog.HeroKey(ev.TargetName)] += cost
	case strings.Contains(ev.ValueName, "dust"):
		a.goldSpentDust[catalog.HeroKey(ev.TargetName)] += cost
	}
}

func (a *Aggregator) onGold(ev events.Combat) {
	if ev.GoldReason == 1 && ev.TargetName != "" {
		v := ev.Value
		if v < 0 {
			v = -v
		}
		a.goldLostByHero[catalog.HeroKey(ev.TargetName)] += v
	}
}

func (a *Aggregator) onDeath(ev events.Combat) {
	if ev.TargetName == "npc_dota_courier" && ev.SourceName != "" {
		a.courierKills[catalog.HeroKey(ev.SourceName)]++
	}
}

func (a *Aggregator) onAbility(ev events.Combat) {
	inf := ev.Inflictor
	if inf == "" || !ev.AttackerHero || ev.AttackerName == "" {
		return
	}
	hk := catalog.HeroKey(ev.AttackerName)
	if _, ok := a.heroTeam[hk]; !ok {
		return
	}
	if talent, ok := catalog.TalentGlobalAbilities[inf]; ok {
		if a.gateOK(catalog.GateTalent, talent, ev.AttackerName, ev.Time) {
			a.purgeCounts[srcKey{hk, catalog.CanonicalName(inf)}]++
		}
		return
	}
	target := ev.TargetName
	if talent, ok := catalog.TalentPurgeAbilities[inf]; ok &&
		ev.TargetHero && !ev.TargetIllusion &&
		target != "" && ev.AttackerName != target && a.heroIsAlly(ev.AttackerName, target) &&
		a.gateOK(catalog.GateTalent, talent, ev.AttackerName, ev.Time) {
		a.purgeCounts[srcKey{hk, catalog.CanonicalName(inf)}]++
	}
}

func (a *Aggregator) onAbilityLevel(hero, ability string, level, t int) {
	if level <= 0 || hero == "" || !catalog.PurgeTalents[ability] {
		return
	}
	k := talentKey{catalog.HeroKey(hero), ability}
	if _, ok := a.talents[k]; !ok {
		a.talents[k] = t
	}
}

func (a *Aggregator) onMiniboss(slot, t int) {
	hero, ok := a.slotHero[slot]
	if !ok {
		return
	}
	hk := catalog.HeroKey(hero)
	if at, ok := a.shardAt[hk]; !ok || t < at {
		a.shardAt[hk] = t
	}
}

func (a *Aggregator) onFirstBlood(slot int) {
	a.firstBloodSlot = slot
}

func (a *Aggregator) Combat(ev events.Combat) {
	switch ev.Type {
	case events.CombatDamage:
		a.onDamage(ev)
	case events.CombatHeal:
		a.onHeal(ev)
	case events.CombatModifierAdd:
		a.onModifier(ev, true)
	case events.CombatModifierRemove:
		a.onModifier(ev, false)
	case events.CombatPurchase:
		a.onPurchase(ev)
	case events.CombatGold:
		a.onGold(ev)
	case events.CombatDeath:
		a.onDeath(ev)
	case events.CombatAbility:
		a.onAbility(ev)
	}
}

func (a *Aggregator) Chat(ev events.Chat) {
	switch ev.Kind {
	case events.ChatFirstBlood:
		a.onFirstBlood(ev.Slot)
	case events.ChatMiniboss:
		a.onMiniboss(ev.Slot, ev.Time)
	}
}

func (a *Aggregator) AbilityLevel(e events.AbilityLevel) {
	a.onAbilityLevel(e.Hero, e.Ability, e.Level, e.Time)
}

func (a *Aggregator) Result() *stats.Match {
	players := append([]events.PlayerInfo(nil), a.roster...)
	if len(players) == 0 {
		for slot, hero := range a.slotHero {
			players = append(players, events.PlayerInfo{Slot: slot, Hero: hero, Team: a.slotTeam[slot]})
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

	a.m.Players = make([]stats.Player, len(players))
	heroKeys := make([]string, len(players))
	for i := range players {
		p := &players[i]
		mp := &a.m.Players[i]
		mp.SteamID = p.SteamID
		mp.PlayerID = p.Slot
		mp.Hero = strings.TrimPrefix(p.Hero, "npc_dota_hero_")
		mp.HeroID = catalog.HeroID(p.Hero)
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
		hk := catalog.HeroKey(p.Hero)
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

	byHero := make(map[string][]*stats.Player, len(a.m.Players))
	for i := range a.m.Players {
		byHero[heroKeys[i]] = append(byHero[heroKeys[i]], &a.m.Players[i])
	}

	for k, v := range a.stunSources {
		for _, mp := range byHero[k.Attacker] {
			mp.StunSources = append(mp.StunSources, stats.SourceEntry{
				Inflictor: k.Inflictor,
				Category:  "stun",
				Duration:  v,
			})
		}
	}
	for k, v := range a.buffSources {
		for _, mp := range byHero[k.Attacker] {
			mp.BuffSources = append(mp.BuffSources, stats.SourceEntry{
				Inflictor: k.Inflictor,
				Category:  k.Category,
				Duration:  math.Round(v),
			})
			mp.BuffDuration += v
		}
	}
	for k, n := range a.purgeCounts {
		for _, mp := range byHero[k.Attacker] {
			mp.BuffSources = append(mp.BuffSources, stats.SourceEntry{
				Inflictor: k.Inflictor,
				Category:  "purge",
				Count:     n,
			})
		}
	}

	specs := []struct {
		kind     catalog.ModKind
		category string
		entries  func(*stats.Player) *[]stats.SourceEntry
		total    func(*stats.Player) *float64
	}{
		{catalog.KindBreak, "break", func(p *stats.Player) *[]stats.SourceEntry { return &p.BreakSources }, func(p *stats.Player) *float64 { return &p.BreakDuration }},
		{catalog.KindFear, "fear", func(p *stats.Player) *[]stats.SourceEntry { return &p.FearSources }, func(p *stats.Player) *float64 { return &p.FearDuration }},
		{catalog.KindRoot, "root", func(p *stats.Player) *[]stats.SourceEntry { return &p.RootSources }, func(p *stats.Player) *float64 { return &p.RootsDuration }},
		{catalog.KindLeash, "leash", func(p *stats.Player) *[]stats.SourceEntry { return &p.LeashSources }, func(p *stats.Player) *float64 { return &p.LeashDuration }},
		{catalog.KindTrap, "trap", func(p *stats.Player) *[]stats.SourceEntry { return &p.TrapSources }, func(p *stats.Player) *float64 { return &p.TrapDuration }},
		{catalog.KindTaunt, "taunt", func(p *stats.Player) *[]stats.SourceEntry { return &p.TauntSources }, func(p *stats.Player) *float64 { return &p.TauntDuration }},
		{catalog.KindSilence, "silence", func(p *stats.Player) *[]stats.SourceEntry { return &p.SilenceSources }, func(p *stats.Player) *float64 { return &p.SilenceDuration }},
		{catalog.KindDisarm, "disarm", func(p *stats.Player) *[]stats.SourceEntry { return &p.DisarmSources }, func(p *stats.Player) *float64 { return &p.DisarmDuration }},
	}
	for _, s := range specs {
		for k, v := range a.durations[s.kind] {
			for _, mp := range byHero[k.Attacker] {
				*s.entries(mp) = append(*s.entries(mp), stats.SourceEntry{
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
			entry := stats.SourceEntry{Inflictor: k.Inflictor, Category: "heal"}
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

	roleInputs := make([]roles.Input, 0, len(a.m.Players))
	for i := range a.m.Players {
		slot := players[i].Slot
		farm, ok := a.roleFarm[slot]
		if !ok || (farm.Networth == 0 && farm.LH == 0 && farm.XP == 0) {
			if iv, ok := a.lastInterval[slot]; ok {
				farm = roles.Farm{Networth: iv.Networth, LH: iv.LastHits, XP: iv.XP}
			}
		}
		roleInputs = append(roleInputs, roles.Input{
			P:     &a.m.Players[i],
			Stats: a.roleSamples[slot],
			Farm:  farm,
		})
	}
	roles.AssignRoles(roleInputs)
	return a.m
}

func sortSources(s []stats.SourceEntry) {
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

func sortHealSources(s []stats.SourceEntry) {
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
