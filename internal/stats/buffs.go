package stats

import (
	"sort"
	"strings"

	"github.com/dotabuff/manta/dota"
)

type buffCat int

const (
	buffNone buffCat = iota
	buffSave
	buffPurge
	buffShield
	buffSpellImmunity
	buffInvisibility
	buffStats
	buffHaste
	buffManaRestore
	buffCatN
)

func buffCatName(cat buffCat) string {
	if cat <= buffNone || int(cat) >= int(buffCatN) {
		return "other"
	}
	return [...]string{
		"", "save", "purge", "shield", "spell_immunity",
		"invisibility", "buff_stats", "buff_haste", "mana_restore",
	}[cat]
}

// buffRule maps a modifier-name substring to a canonical category and display
// name. Canonical names keep the output stable across modifier variants
// (item_glimmer_cape_fade and item_glimmer_cape_invis both report
// "item_glimmer_cape"). Rules are ordered: more specific substrings first.
// Heal and mana-restore amounts are tracked from their own events, so this
// catalog covers only the duration-based action buffs; passive auras are
// filtered at runtime via the combat log's aura_modifier flag, and self-casts
// via target == attacker.
type buffRule struct {
	substr string
	cat    buffCat
	name   string
}

var buffRules = []buffRule{
	// save
	{"shallow_grave", buffSave, "dazzle_shallow_grave"},
	{"false_promise", buffSave, "oracle_false_promise"},
	{"cold_embrace", buffSave, "winter_wyvern_cold_embrace"},
	{"guardian_angel", buffSave, "omniknight_guardian_angel"},
	{"borrowed_time", buffSave, "abaddon_borrowed_time"},
	{"reincarnation_scepter", buffSave, "skeleton_king_reincarnation_scepter"},
	{"astral_imprisonment", buffSave, "obsidian_destroyer_astral_imprisonment"},
	{"telekinesis", buffSave, "rubick_telekinesis"},
	{"decrepify", buffSave, "pugna_decrepify"},
	{"the_box", buffSave, "ringmaster_the_box"},
	{"strongman_tonic", buffSave, "ringmaster_strongman_tonic"},
	{"snowball", buffSave, "tusk_snowball"},
	{"aeon_disk", buffSave, "item_aeon_disk"},
	{"cyclone", buffSave, "item_cyclone"},
	{"forcestaff", buffSave, "item_force_staff"},
	{"hurricane_pike", buffSave, "item_hurricane_pike"},

	// purge
	{"guardian_greaves", buffPurge, "item_guardian_greaves"},
	{"press_the_attack", buffPurge, "legion_commander_press_the_attack"},
	{"purifying_flames", buffPurge, "oracle_purifying_flames"},
	{"omniknight_purification", buffPurge, "omniknight_purification"},
	{"lotus_orb", buffPurge, "item_lotus_orb"},

	// shield
	{"aphotic_shield", buffShield, "abaddon_aphotic_shield"},
	{"pavise_shield", buffShield, "item_pavise_shield"},
	{"frost_shield", buffShield, "lich_frost_shield"},
	{"ash_legion_shield", buffShield, "item_ash_legion_shield"},
	{"bodyguarded", buffShield, "marci_bodyguard"},
	{"item_pipe", buffShield, "item_pipe_of_insight"},

	// spell immunity
	{"omniknight_repel", buffSpellImmunity, "omniknight_repel"},

	// invisibility
	{"glimmer_cape", buffInvisibility, "item_glimmer_cape"},
	{"shadow_amulet", buffInvisibility, "item_shadow_amulet"},
	{"smoke_of_deceit", buffInvisibility, "smoke_of_deceit"},
	{"moonlight_shadow", buffInvisibility, "mirana_moonlight_shadow"},

	// buff stats
	{"solar_crest", buffStats, "item_solar_crest"},
	{"magnataur_empower", buffStats, "magnataur_empower"},
	{"ogre_magi_bloodlust", buffStats, "ogre_magi_bloodlust"},

	// buff haste
	{"drum_of_endurance", buffHaste, "item_drum_of_endurance"},
	{"centaur_stampede", buffHaste, "centaur_stampede"},

	// mana restore (amount is not exposed; uptime is the proxy). Arcane boots
	// are NOT listed: their modifier is self-targeted and hidden on every
	// patch observed, so the per-ally mana grant is not trackable.
	{"chakra_magic", buffManaRestore, "keeper_of_the_light_chakra_magic"},
}

// classifyBuff returns the buff category and canonical display name for a
// combat-log name, or buffNone when the modifier is not tracked.
func classifyBuff(name string) (buffCat, string) {
	l := strings.ToLower(strings.TrimPrefix(name, "modifier_"))
	for _, r := range buffRules {
		if strings.Contains(l, r.substr) {
			return r.cat, r.name
		}
	}
	return buffNone, ""
}

type buffModKey struct {
	cat buffCat
	inf string
	tgt uint32
}

type buffInf struct {
	cat    buffCat
	uptime float64
	count  int
}

// buffTrack holds paired MODIFIER_ADD→REMOVE uptime for ally-applied buffs,
// one instance per caster (stored on the caster's xpStats).
type buffTrack struct {
	byInf   map[string]*buffInf
	pending map[buffModKey]int
}

func (b *buffTrack) track(cat buffCat, inf string) *buffInf {
	if b.byInf == nil {
		b.byInf = make(map[string]*buffInf)
	}
	t, ok := b.byInf[inf]
	if !ok {
		t = &buffInf{cat: cat}
		b.byInf[inf] = t
	}
	return t
}

// totalUptime sums the tracked uptime across all modifiers; it is the
// per-player buffs_duration total.
func (b buffTrack) totalUptime() float64 {
	var sum float64
	for _, t := range b.byInf {
		sum += t.uptime
	}
	return sum
}

func (b *buffTrack) add(cat buffCat, inf string, tgt uint32) {
	if cat == buffNone {
		return
	}
	if b.pending == nil {
		b.pending = make(map[buffModKey]int)
	}
	k := buffModKey{cat, inf, tgt}
	b.pending[k]++
	b.track(cat, inf)
}

func (b *buffTrack) remove(cat buffCat, inf string, tgt uint32, elapsed float64) {
	if cat == buffNone || elapsed <= 0 {
		return
	}
	k := buffModKey{cat, inf, tgt}
	if b.pending == nil || b.pending[k] <= 0 {
		return
	}
	b.pending[k]--
	t := b.track(cat, inf)
	t.uptime += elapsed
	t.count++
}

// trackModifierBuff records a positive effect applied by a hero from the
// ally-capable catalog, whether on an ally or on itself (a self-cast of an
// ally-capable ability is a real buff, e.g. Abaddon shielding himself). Aura
// and hidden modifiers are excluded: auras are passive, hidden ones are the
// self-only/passive variants (Wraith King's passive reincarnation, greaves
// self-buff) whose names the catalog rules happen to cover. The team fields
// are populated on every MODIFIER event on current patches; when either is
// missing the event is skipped (conservative, avoids counting enemy-cast
// debuff variants of the same abilities).
func (a *aggregator) trackModifierBuff(e *dota.CMsgDOTACombatLogEntry) {
	if a.lookup == nil {
		return
	}
	if e.GetIsTargetIllusion() {
		return
	}
	if e.GetAuraModifier() || e.GetHiddenModifier() {
		return
	}
	if at, tt := e.GetAttackerTeam(), e.GetTargetTeam(); at == 0 || at != tt {
		return
	}
	name, ok := a.lookup("CombatLogNames", int32(e.GetInflictorName()))
	if !ok || name == "" {
		return
	}
	cat, inf := classifyBuff(name)
	if cat == buffNone {
		return
	}
	x := a.xp(e.GetAttackerName())
	switch e.GetType() {
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD:
		if !e.GetIsTargetHero() {
			return
		}
		x.buff.add(cat, inf, e.GetTargetName())
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE:
		if elapsed := e.GetModifierElapsedDuration(); elapsed > 0 {
			x.buff.remove(cat, inf, e.GetTargetName(), float64(elapsed))
		}
	}
}

// trackAllyHeal credits active heals cast by a hero on same-team heroes,
// including on itself (self-casts of ally-capable abilities heal their owner
// too, e.g. Abaddon's shield or Dazzle's shadow wave), excluding lifesteal and
// regen sources (passive/aura heals). The combat log carries the ability name
// on every HEAL event, so the credit is attributed per ability.
func (a *aggregator) trackAllyHeal(e *dota.CMsgDOTACombatLogEntry) {
	if !e.GetIsTargetHero() || e.GetIsTargetIllusion() {
		return
	}
	if e.GetHealFromLifesteal() || e.GetHealFromRegen() {
		return
	}
	if at, tt := e.GetAttackerTeam(), e.GetTargetTeam(); at == 0 || at != tt {
		return
	}
	value := float64(e.GetValue())
	x := a.xp(e.GetAttackerName())
	x.buffHeal += value
	x.buffHealN++

	inf := "heal"
	if a.lookup != nil {
		if n, ok := a.lookup("CombatLogNames", int32(e.GetInflictorName())); ok && n != "" {
			inf = normInf(n)
		}
	}
	if x.buffHealInf == nil {
		x.buffHealInf = make(map[string]float64)
		x.buffHealInfN = make(map[string]int)
	}
	x.buffHealInf[inf] += value
	x.buffHealInfN[inf]++
}

// buildBuffSources assembles the per-player BuffSources from the caster's
// aggregates: duration buffs as uptime seconds + application count, heals as
// total HP + heal-event count. Sorted by value descending like stun_sources.
func buildBuffSources(x xpStats) []BuffSource {
	var out []BuffSource
	for inf, t := range x.buff.byInf {
		if t.uptime <= 0 {
			continue
		}
		out = append(out, BuffSource{
			Inflictor: inf,
			Category:  buffCatName(t.cat),
			Value:     shortestFloat32(t.uptime),
			Count:     t.count,
		})
	}
	for inf, v := range x.buffHealInf {
		out = append(out, BuffSource{
			Inflictor: inf,
			Category:  "heal",
			Value:     shortestFloat32(v),
			Count:     x.buffHealInfN[inf],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Inflictor < out[j].Inflictor
	})
	if len(out) == 0 {
		return nil
	}
	return out
}
