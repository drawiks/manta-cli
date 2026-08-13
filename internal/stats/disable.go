package stats

import (
	"sort"
	"strings"

	"github.com/dotabuff/manta/dota"
)

type disableCat int

const (
	disableNone disableCat = iota
	disableFear
	disableSleep
	disableRoot
	disableTaunt
	disableSilence
	disableBreak
	disableMute
	disableBanish
	disableLeash
	disableTrap
	disableCatN
)

type infTrack struct {
	cat                  disableCat
	addStun, removeEl    float64
	addN, addStunN, removeN int
}

type modKey struct {
	cat disableCat
	xp  uint32
	inf string
	tgt uint32
}

func normInf(name string) string {
	return strings.TrimPrefix(strings.ToLower(name), "modifier_")
}

func infKey(cat disableCat, name string) string {
	key := normInf(name)
	switch cat {
	case disableSleep, disableTrap, disableLeash:
		for _, suf := range []string{
			"_healing_aura", "_healing", "_aura", "_debuff", "_leash", "_spear",
		} {
			key = strings.TrimSuffix(key, suf)
		}
	case disableFear, disableSilence:
		for _, suf := range []string{"_aura", "_debuff"} {
			key = strings.TrimSuffix(key, suf)
		}
	case disableTaunt:
		for _, suf := range []string{"_armor", "_buff"} {
			key = strings.TrimSuffix(key, suf)
		}
	}
	return key
}

// disableTrack holds supplemental disable uptime not fully credited in m_fStuns.
type disableTrack struct {
	byInf   map[string]*infTrack
	pending map[modKey]int
}

func (d *disableTrack) track(cat disableCat, inf string) *infTrack {
	key := infKey(cat, inf)
	if d.byInf == nil {
		d.byInf = make(map[string]*infTrack)
	}
	t, ok := d.byInf[key]
	if !ok {
		t = &infTrack{cat: cat}
		d.byInf[key] = t
	}
	return t
}

func (d *disableTrack) add(cat disableCat, xp uint32, inf string, tgt uint32, stun float64) {
	if cat == disableNone {
		return
	}
	if d.pending == nil {
		d.pending = make(map[modKey]int)
	}
	t := d.track(cat, inf)
	if stun > 0 {
		t.addStun += stun
		t.addStunN++
	}
	t.addN++
	k := modKey{cat, xp, infKey(cat, inf), tgt}
	d.pending[k]++
}

func (d *disableTrack) remove(cat disableCat, xp uint32, inf string, tgt uint32, elapsed float64) {
	if cat == disableNone || elapsed <= 0 {
		return
	}
	k := modKey{cat, xp, infKey(cat, inf), tgt}
	if d.pending == nil || d.pending[k] <= 0 {
		return
	}
	d.pending[k]--
	t := d.track(cat, inf)
	t.removeEl += elapsed
	t.removeN++
}

func infCredit(t *infTrack) float64 {
	if t.removeN <= 0 || t.removeEl <= 0 {
		return 0
	}
	// Silence without ADD stun_duration is never in m_fStuns; credit paired uptime.
	if t.addStunN <= 0 {
		if t.cat == disableSilence {
			return scalePaired(t, t.removeEl)
		}
		return 0
	}
	gap := t.removeEl - t.addStun
	if gap <= 0 {
		return 0
	}
	return scalePaired(t, gap)
}

func scalePaired(t *infTrack, sec float64) float64 {
	if t.cat == disableSleep || t.cat == disableTrap || t.cat == disableLeash ||
		t.cat == disableFear || t.cat == disableSilence {
		scaleN := t.addStunN
		if scaleN == 0 {
			scaleN = t.addN
		}
		if t.removeN > scaleN && scaleN > 0 {
			sec *= float64(scaleN) / float64(t.removeN)
		}
	}
	return sec
}

func (d disableTrack) supplementalDisable() float64 {
	var extra float64
	for _, t := range d.byInf {
		extra += infCredit(t)
	}
	return extra
}

func catName(cat disableCat) string {
	if cat <= disableNone || int(cat) >= int(disableCatN) {
		return "other"
	}
	return [...]string{
		"", "fear", "sleep", "root", "taunt", "silence",
		"break", "mute", "banish", "leash", "trap",
	}[cat]
}

type stunBuildInput struct {
	base       float64
	scoreboard bool
	stunByInf  map[string]float64
	disable    disableTrack
}

func buildStunSources(in stunBuildInput) []StunSource {
	var out []StunSource

	decompose := in.scoreboard && in.base > 0 && trackedAddStun(in.disable) <= in.base+0.05

	if decompose {
		var trackedAdd float64
		for inf, t := range in.disable.byInf {
			credit := infCredit(t)
			sec := disableSourceSec(t, credit)
			if sec <= 0 {
				continue
			}
			if t.addStunN > 0 {
				trackedAdd += t.addStun
			}
			out = append(out, StunSource{
				Inflictor: inf,
				Category:  catName(t.cat),
				Seconds:   shortestFloat32(sec),
			})
		}
		if rem := in.base - trackedAdd; rem > 0.01 {
			out = append(out, StunSource{
				Inflictor: "scoreboard",
				Category:  "stun",
				Seconds:   shortestFloat32(rem),
			})
		}
	} else {
		switch {
		case in.scoreboard && in.base > 0:
			out = append(out, StunSource{
				Inflictor: "scoreboard",
				Category:  "stun",
				Seconds:   shortestFloat32(in.base),
			})
		case len(in.stunByInf) > 0:
			for inf, sec := range in.stunByInf {
				if sec <= 0 {
					continue
				}
				if classifyDisable(inf) != disableNone {
					continue
				}
				out = append(out, StunSource{
					Inflictor: inf,
					Category:  "stun",
					Seconds:   shortestFloat32(sec),
				})
			}
		case in.base > 0:
			out = append(out, StunSource{
				Inflictor: "combat_log",
				Category:  "stun",
				Seconds:   shortestFloat32(in.base),
			})
		}

		mergeStunByInf := !in.scoreboard
		for inf, t := range in.disable.byInf {
			credit := infCredit(t)
			sec := credit
			if mergeStunByInf {
				if extra, ok := in.stunByInf[inf]; ok {
					sec += extra
				}
			} else if t.addStunN > 0 {
				sec = disableSourceSec(t, credit)
			}
			if sec <= 0 {
				continue
			}
			out = append(out, StunSource{
				Inflictor: inf,
				Category:  catName(t.cat),
				Seconds:   shortestFloat32(sec),
			})
		}
		if mergeStunByInf {
			for inf, sec := range in.stunByInf {
				if sec <= 0 || classifyDisable(inf) == disableNone {
					continue
				}
				if _, ok := in.disable.byInf[inf]; ok {
					continue
				}
				out = append(out, StunSource{
					Inflictor: inf,
					Category:  catName(classifyDisable(inf)),
					Seconds:   shortestFloat32(sec),
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Seconds != out[j].Seconds {
			return out[i].Seconds > out[j].Seconds
		}
		return out[i].Inflictor < out[j].Inflictor
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func trackedAddStun(d disableTrack) float64 {
	var sum float64
	for _, t := range d.byInf {
		sum += t.addStun
	}
	return sum
}

// disableSourceSec is paired uptime (ADD→REMOVE) attributed to one inflictor line.
func disableSourceSec(t *infTrack, credit float64) float64 {
	if credit <= 0 {
		return 0
	}
	if t.addStunN > 0 {
		return t.addStun + credit
	}
	return credit
}

func classifyDisable(inflictor string) disableCat {
	l := strings.ToLower(inflictor)

	if isHexModifier(l) || isClassicStunModifier(l) {
		return disableNone
	}

	switch {
	case strings.Contains(l, "crippling_fear"):
		// Night Stalker Crippling Fear silences (MODIFIER_STATE_SILENCED), not fear.
		return disableSilence

	case strings.Contains(l, "berserkers_call_armor"):
		// Self armor buff during Berserker's Call, not enemy taunt.
		return disableNone

	case strings.Contains(l, "wave_of_silence_buff"):
		// Self buff on Drow during Wave of Silence cast.
		return disableNone

	case strings.Contains(l, "fear") || strings.Contains(l, "terrorize") ||
		strings.Contains(l, "sinister_gaze") || strings.Contains(l, "requiem_fear"):
		return disableFear

	case strings.Contains(l, "sleep") || strings.Contains(l, "song_of_the_siren") ||
		strings.Contains(l, "nightmare") || strings.Contains(l, "echo_stomp") ||
		strings.Contains(l, "nocturnal_capture"):
		return disableSleep

	case strings.Contains(l, "cyclone") || strings.Contains(l, "eul") ||
		strings.Contains(l, "wind_waker") || strings.Contains(l, "tornado") ||
		strings.Contains(l, "astral_imprisonment") || strings.Contains(l, "disruption") ||
		strings.Contains(l, "banish"):
		return disableBanish

	case strings.Contains(l, "taunt") || strings.Contains(l, "berserker") ||
		strings.Contains(l, "duel") || strings.Contains(l, "winters_curse") ||
		strings.Contains(l, "little_friends"):
		return disableTaunt

	case isSilenceModifier(l):
		return disableSilence

	case isBreakModifier(l):
		return disableBreak

	case strings.Contains(l, "mute") || strings.Contains(l, "heavens_halberd"):
		return disableMute

	case strings.Contains(l, "kinetic_field") || strings.Contains(l, "static_storm") ||
		strings.Contains(l, "arena_of_blood"):
		return disableTrap

	case hasLeash(l):
		return disableLeash

	case strings.Contains(l, "root") || strings.Contains(l, "ensnare") ||
		strings.Contains(l, "frostbite") || strings.Contains(l, "entangle") ||
		strings.Contains(l, "overgrowth") || strings.Contains(l, "earthbind") ||
		strings.Contains(l, "searing_chains") ||
		strings.Contains(l, "pit_of_malice") || strings.Contains(l, "eternal_chains") ||
		strings.Contains(l, "cold_feet") || strings.Contains(l, "ice_wall") ||
		strings.Contains(l, "venomous_gale") || strings.Contains(l, "bramble") ||
		strings.Contains(l, "fortune") && strings.Contains(l, "end"):
		return disableRoot

	default:
		return disableNone
	}
}

func isSilenceModifier(l string) bool {
	if normInf(l) == "silence" {
		return false
	}
	return strings.Contains(l, "silence") || strings.Contains(l, "arcane_curse") ||
		strings.Contains(l, "curse_of_the_silent")
}

func isBreakModifier(l string) bool {
	if normInf(l) == "break" {
		return false
	}
	return strings.Contains(l, "passive_disabled") || strings.Contains(l, "_break")
}

func hasLeash(l string) bool {
	if strings.Contains(l, "unleash") {
		return false
	}
	return strings.Contains(l, "_leash") || strings.Contains(l, "dream_coil") ||
		strings.Contains(l, "pounce")
}

func isHexModifier(l string) bool {
	return strings.Contains(l, "hex") || strings.Contains(l, "voodoo") ||
		strings.Contains(l, "sheep")
}

func isClassicStunModifier(l string) bool {
	switch {
	case strings.Contains(l, "stunned"):
		return true
	case strings.Contains(l, "bash"):
		return true
	case strings.Contains(l, "hammer"):
		return true
	case strings.Contains(l, "crush"):
		return true
	case strings.Contains(l, "ravage"):
		return true
	case strings.Contains(l, "black_hole"):
		return true
	case strings.Contains(l, "shackle"):
		return true
	case strings.Contains(l, "polarity"):
		return true
	case strings.Contains(l, "fiends_grip"):
		return true
	case strings.Contains(l, "chronosphere"):
		return true
	case strings.Contains(l, "impale"):
		return true
	case strings.Contains(l, "fissure"):
		return true
	case strings.Contains(l, "spike"):
		return true
	case strings.Contains(l, "requiem") && !strings.Contains(l, "fear"):
		return true
	default:
		return strings.Contains(l, "stun") && !strings.Contains(l, "no_stun")
	}
}

func (a *aggregator) trackModifierDisable(e *dota.CMsgDOTACombatLogEntry) {
	if a.lookup == nil {
		return
	}
	if e.GetIsTargetIllusion() {
		return
	}
	name, ok := a.lookup("CombatLogNames", int32(e.GetInflictorName()))
	if !ok || name == "" {
		return
	}
	cat := classifyDisable(name)
	if cat == disableNone {
		return
	}
	// Self-applied modifiers (armor during taunt, eul self, etc.) are not
	// disable time inflicted on enemies.
	if e.GetAttackerName() == e.GetTargetName() {
		return
	}
	xp := e.GetAttackerName()
	tgt := e.GetTargetName()
	x := a.xp(xp)
	switch e.GetType() {
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_ADD:
		if !e.GetIsTargetHero() {
			return
		}
		x.disable.add(cat, xp, name, tgt, float64(e.GetStunDuration()))
	case dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MODIFIER_REMOVE:
		if elapsed := e.GetModifierElapsedDuration(); elapsed > 0 {
			x.disable.remove(cat, xp, name, tgt, float64(elapsed))
		}
	}
}

func (x *xpStats) addStunInf(inflictor string, stun float64) {
	if stun <= 0 {
		return
	}
	if x.stunByInf == nil {
		x.stunByInf = make(map[string]float64)
	}
	x.stunByInf[normInf(inflictor)] += stun
}
