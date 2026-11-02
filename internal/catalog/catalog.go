package catalog

import (
	"regexp"
	"strings"
)

// HeroMap maps a replay unit name to the Valve hero id. The table is game
// data, kept in sync with the canonical npc_dota_hero_* names.
var HeroMap = map[string]int{
	"npc_dota_hero_antimage":            1,
	"npc_dota_hero_axe":                 2,
	"npc_dota_hero_bane":                3,
	"npc_dota_hero_bloodseeker":         4,
	"npc_dota_hero_crystal_maiden":      5,
	"npc_dota_hero_drow_ranger":         6,
	"npc_dota_hero_earthshaker":         7,
	"npc_dota_hero_juggernaut":          8,
	"npc_dota_hero_mirana":              9,
	"npc_dota_hero_morphling":           10,
	"npc_dota_hero_nevermore":           11,
	"npc_dota_hero_phantom_lancer":      12,
	"npc_dota_hero_puck":                13,
	"npc_dota_hero_pudge":               14,
	"npc_dota_hero_razor":               15,
	"npc_dota_hero_sand_king":           16,
	"npc_dota_hero_storm_spirit":        17,
	"npc_dota_hero_sven":                18,
	"npc_dota_hero_tiny":                19,
	"npc_dota_hero_vengefulspirit":      20,
	"npc_dota_hero_windrunner":          21,
	"npc_dota_hero_zuus":                22,
	"npc_dota_hero_kunkka":              23,
	"npc_dota_hero_lina":                25,
	"npc_dota_hero_lion":                26,
	"npc_dota_hero_shadow_shaman":       27,
	"npc_dota_hero_slardar":             28,
	"npc_dota_hero_tidehunter":          29,
	"npc_dota_hero_witch_doctor":        30,
	"npc_dota_hero_lich":                31,
	"npc_dota_hero_riki":                32,
	"npc_dota_hero_enigma":              33,
	"npc_dota_hero_tinker":              34,
	"npc_dota_hero_sniper":              35,
	"npc_dota_hero_necrolyte":           36,
	"npc_dota_hero_warlock":             37,
	"npc_dota_hero_beastmaster":         38,
	"npc_dota_hero_queenofpain":         39,
	"npc_dota_hero_venomancer":          40,
	"npc_dota_hero_faceless_void":       41,
	"npc_dota_hero_skeleton_king":       42,
	"npc_dota_hero_death_prophet":       43,
	"npc_dota_hero_phantom_assassin":    44,
	"npc_dota_hero_pugna":               45,
	"npc_dota_hero_templar_assassin":    46,
	"npc_dota_hero_viper":               47,
	"npc_dota_hero_luna":                48,
	"npc_dota_hero_dragon_knight":       49,
	"npc_dota_hero_dazzle":              50,
	"npc_dota_hero_rattletrap":          51,
	"npc_dota_hero_leshrac":             52,
	"npc_dota_hero_furion":              53,
	"npc_dota_hero_life_stealer":        54,
	"npc_dota_hero_dark_seer":           55,
	"npc_dota_hero_clinkz":              56,
	"npc_dota_hero_omniknight":          57,
	"npc_dota_hero_enchantress":         58,
	"npc_dota_hero_huskar":              59,
	"npc_dota_hero_night_stalker":       60,
	"npc_dota_hero_broodmother":         61,
	"npc_dota_hero_bounty_hunter":       62,
	"npc_dota_hero_weaver":              63,
	"npc_dota_hero_jakiro":              64,
	"npc_dota_hero_batrider":            65,
	"npc_dota_hero_chen":                66,
	"npc_dota_hero_spectre":             67,
	"npc_dota_hero_ancient_apparition":  68,
	"npc_dota_hero_doom_bringer":        69,
	"npc_dota_hero_ursa":                70,
	"npc_dota_hero_spirit_breaker":      71,
	"npc_dota_hero_gyrocopter":          72,
	"npc_dota_hero_alchemist":           73,
	"npc_dota_hero_invoker":             74,
	"npc_dota_hero_silencer":            75,
	"npc_dota_hero_obsidian_destroyer":  76,
	"npc_dota_hero_lycan":               77,
	"npc_dota_hero_brewmaster":          78,
	"npc_dota_hero_shadow_demon":        79,
	"npc_dota_hero_lone_druid":          80,
	"npc_dota_hero_chaos_knight":        81,
	"npc_dota_hero_meepo":               82,
	"npc_dota_hero_treant":              83,
	"npc_dota_hero_ogre_magi":           84,
	"npc_dota_hero_undying":             85,
	"npc_dota_hero_rubick":              86,
	"npc_dota_hero_disruptor":           87,
	"npc_dota_hero_nyx_assassin":        88,
	"npc_dota_hero_naga_siren":          89,
	"npc_dota_hero_keeper_of_the_light": 90,
	"npc_dota_hero_wisp":                91,
	"npc_dota_hero_visage":              92,
	"npc_dota_hero_slark":               93,
	"npc_dota_hero_medusa":              94,
	"npc_dota_hero_troll_warlord":       95,
	"npc_dota_hero_centaur":             96,
	"npc_dota_hero_magnataur":           97,
	"npc_dota_hero_shredder":            98,
	"npc_dota_hero_bristleback":         99,
	"npc_dota_hero_tusk":                100,
	"npc_dota_hero_skywrath_mage":       101,
	"npc_dota_hero_abaddon":             102,
	"npc_dota_hero_elder_titan":         103,
	"npc_dota_hero_legion_commander":    104,
	"npc_dota_hero_techies":             105,
	"npc_dota_hero_ember_spirit":        106,
	"npc_dota_hero_earth_spirit":        107,
	"npc_dota_hero_abyssal_underlord":   108,
	"npc_dota_hero_terrorblade":         109,
	"npc_dota_hero_phoenix":             110,
	"npc_dota_hero_oracle":              111,
	"npc_dota_hero_winter_wyvern":       112,
	"npc_dota_hero_arc_warden":          113,
	"npc_dota_hero_monkey_king":         114,
	"npc_dota_hero_dark_willow":         119,
	"npc_dota_hero_pangolier":           120,
	"npc_dota_hero_grimstroke":          121,
	"npc_dota_hero_hoodwink":            123,
	"npc_dota_hero_void_spirit":         126,
	"npc_dota_hero_snapfire":            128,
	"npc_dota_hero_mars":                129,
	"npc_dota_hero_ringmaster":          131,
	"npc_dota_hero_dawnbreaker":         135,
	"npc_dota_hero_marci":               136,
	"npc_dota_hero_primal_beast":        137,
	"npc_dota_hero_muerta":              138,
	"npc_dota_hero_kez":                 145,
	"npc_dota_hero_largo":               155,
}

// heroIDByKey lets us resolve ids when the replay's unit name uses the
// dashed variant (queenof_pain) instead of the canonical slug (queenofpain).
var heroIDByKey = func() map[string]int {
	m := make(map[string]int, len(HeroMap))
	for name, id := range HeroMap {
		m[HeroKey(name)] = id
	}
	return m
}()

// BuffCategories maps a cleaned modifier name to the positive-effect category
// reported in buff_sources.
var BuffCategories = map[string]string{
	"dazzle_shallow_grave":                   "save",
	"oracle_false_promise":                   "save",
	"omniknight_guardian_angel":              "save",
	"abaddon_borrowed_time":                  "save",
	"obsidian_destroyer_astral_imprisonment": "save",
	"pugna_decrepify":                        "save",
	"underlord_dark_rift":                    "save",
	"ringmaster_strongman_tonic":             "save",
	"tusk_snowball":                          "save",
	"ringmaster_the_box_buff":                "save",
	"eul_cyclone":                            "save",
	"item_force_staff":                       "save",
	"item_hurricane_pike":                    "save",
	"disperser_movespeed_buff":               "save",

	"abaddon_aphotic_shield": "shield",
	"item_pavise":            "shield",
	"lich_frost_shield":      "shield",
	"marci_bodyguard":        "shield",
	"item_pipe_of_insight":   "shield",
	"ogre_magi_smash_buff":   "shield",

	"omniknight_repel": "spell_immunity",

	"item_glimmer_cape_fade":  "invisibility",
	"item_shadow_amulet_fade": "invisibility",
	"mirana_moonlight_shadow": "invisibility",
	"bounty_hunter_wind_walk": "invisibility",

	"ogre_magi_bloodlust":                      "buff_stats",
	"treant_living_armor":                      "buff_stats",
	"marci_companion_run_ally_movespeed":       "buff_stats",
	"marci_bodyguarded":                        "buff_stats",
	"io_overcharge":                            "buff_stats",
	"dark_seer_ion_shell":                      "buff_stats",
	"chen_divine_favor_armor":                  "buff_stats",
	"empower":                                  "buff_stats",
	"item_solar_crest_armor_addition":          "buff_stats",
	"item_medallion_of_courage_armor_addition": "buff_stats",
	"alchemist_berserk_potion":                 "buff_stats",

	"item_drum":        "buff_haste",
	"centaur_stampede": "buff_haste",
	"dark_seer_surge":  "buff_haste",
}

// PurgeModifiers always dispel allies and are counted per application.
var PurgeModifiers = map[string]bool{
	"item_guardian_greaves":             true,
	"legion_commander_press_the_attack": true,
	"oracle_purifying_flames":           true,
	"item_lotus_orb_active":             true,
	"alchemist_berserk_potion":          true,
	"shadow_demon_purge_slow":           true,
}

// ShardPurgeModifiers only dispel after the caster bought Aghanim's Shard.
var ShardPurgeModifiers = map[string]bool{
	"grimstroke_spirit_walk_buff": true,
}

// TalentPurgeModifiers only dispel after the caster learns the talent.
var TalentPurgeModifiers = map[string]string{
	"ringmaster_the_box_buff": "special_bonus_unique_ringmaster_the_box_strong_dispel_and_flying",
}

// TalentPurgeAbilities are ally-targeted casts that only dispel after the
// caster learns the talent.
var TalentPurgeAbilities = map[string]string{
	"keeper_of_the_light_chakra_magic": "special_bonus_unique_keeper_of_the_light_14",
}

// TalentGlobalAbilities are global casts counted once per cast, gated on the
// caster's talent.
var TalentGlobalAbilities = map[string]string{
	"chen_hand_of_god": "special_bonus_unique_chen_12",
}

// PurgeTalents is the set of talent names tracked for purge gating.
var PurgeTalents = map[string]bool{
	"special_bonus_unique_keeper_of_the_light_14":                      true,
	"special_bonus_unique_chen_12":                                     true,
	"special_bonus_unique_ringmaster_the_box_strong_dispel_and_flying": true,
}

var InstantHealItems = map[string]bool{
	"item_holy_locket":     true,
	"item_famango":         true,
	"item_great_famango":   true,
	"item_greater_famango": true,
}

var ItemWardCosts = map[string]int{
	"item_smoke_of_deceit": 50,
	"item_dust":            80,
}

type ModKind uint8

const (
	KindNone ModKind = iota
	KindBreak
	KindFear
	KindRoot
	KindLeash
	KindTrap
	KindTaunt
	KindSilence
	KindDisarm
	KindHeal
	KindBuff
	KindCount
)

type PurgeGate uint8

const (
	GateNone PurgeGate = iota
	GateShard
	GateTalent
)

type ModClass struct {
	Kind     ModKind
	Category string
	Clean    string
	Purge    bool
	Gate     PurgeGate
	Talent   string
}

var suffixStrips = []string{
	"_healing_aura", "_healing", "_aura", "_debuff",
	"_leash", "_spear", "_fade", "_buff",
	"_armor", "_active",
}

func stripModifier(inf string) string {
	return strings.TrimPrefix(inf, "modifier_")
}

func CanonicalName(clean string) string {
	for _, s := range suffixStrips {
		clean = strings.TrimSuffix(clean, s)
	}
	return clean
}

func ClassifyModifier(inf string) ModClass {
	rawClean := stripModifier(inf)
	c := ModClass{Clean: CanonicalName(rawClean), Purge: PurgeModifiers[rawClean]}
	if ShardPurgeModifiers[rawClean] {
		c.Purge = true
		c.Gate = GateShard
	}
	if talent, ok := TalentPurgeModifiers[rawClean]; ok {
		c.Purge = true
		c.Gate = GateTalent
		c.Talent = talent
	}
	switch {
	case classifyBreak(inf):
		c.Kind = KindBreak
	case classifyFear(inf):
		c.Kind = KindFear
	case classifyRoot(inf):
		c.Kind = KindRoot
	case classifyLeash(inf):
		c.Kind = KindLeash
	case classifyTrap(inf):
		c.Kind = KindTrap
	case classifyTaunt(inf):
		c.Kind = KindTaunt
	case classifySilence(inf):
		c.Kind = KindSilence
	case classifyDisarm(inf):
		c.Kind = KindDisarm
	case classifyHeal(inf):
		c.Kind = KindHeal
	default:
		if cat := BuffCategories[rawClean]; cat != "" {
			c.Kind = KindBuff
			c.Category = cat
		}
	}
	return c
}

func classifyBreak(inf string) bool {
	l := strings.ToLower(inf)
	if l == "break" || l == "modifier_break" {
		return true
	}
	return strings.Contains(l, "silver_edge_debuff") ||
		strings.Contains(l, "vendetta_break") ||
		strings.Contains(l, "angels_demise_break") ||
		strings.Contains(l, "doom_bringer_doom_break") ||
		strings.Contains(l, "viper_strike_debuff") ||
		strings.Contains(l, "sharpshooter_debuff") ||
		strings.Contains(l, "fan_of_knives") ||
		strings.Contains(l, "berserker_troll_break")
}

func classifyFear(inf string) bool {
	l := strings.ToLower(inf)
	if strings.Contains(l, "crippling_fear") {
		return false
	}
	return strings.Contains(l, "fear") ||
		strings.Contains(l, "terrorize") ||
		strings.Contains(l, "sinister_gaze") ||
		strings.Contains(l, "savage_roar") ||
		strings.Contains(l, "dead_shot") ||
		strings.Contains(l, "tame_the_beasts") ||
		strings.Contains(l, "terror_wave") ||
		strings.Contains(l, "aether_remnant") ||
		strings.Contains(l, "will_o_wisp") ||
		strings.Contains(l, "wheel_of_wonder")
}

func classifyRoot(inf string) bool {
	l := strings.ToLower(inf)
	return strings.Contains(l, "root") ||
		strings.Contains(l, "frostbite") ||
		strings.Contains(l, "searing_chains") ||
		strings.Contains(l, "earthbind") ||
		strings.Contains(l, "ensnare") ||
		strings.Contains(l, "entangle") ||
		strings.Contains(l, "fortune") ||
		strings.Contains(l, "spinnerssnare") ||
		strings.Contains(l, "atos")
}

func classifyLeash(inf string) bool {
	l := strings.ToLower(inf)
	if strings.Contains(l, "puck_coiled") && !strings.Contains(l, "break") {
		return true
	}
	return strings.Contains(l, "pounce_leash") ||
		strings.Contains(l, "soul_chain")
}

func classifyTrap(inf string) bool {
	l := strings.ToLower(inf)
	return strings.Contains(l, "kinetic_field") ||
		strings.Contains(l, "arena_of_blood_leash")
}

func classifyTaunt(inf string) bool {
	l := strings.ToLower(inf)
	return strings.Contains(l, "berserkers_call") ||
		strings.Contains(l, "little_friends") ||
		strings.Contains(l, "winters_curse") ||
		strings.Contains(l, "legion_commander_duel") ||
		strings.Contains(l, "life_break_taunt")
}

var silenceSubstrings = []string{
	"crippling_fear",
	"static_storm",
	"ancient_seal",
	"doom_bringer_doom",
	"smoke_screen",
	"geomagnetic_grip_debuff",
	"bloodthorn",
	"orchid",
	"ink_creature_debuff",
	"flux_alone",
	"inner_fire_disarm",
	"silencer_last_word",
}

func classifySilence(inf string) bool {
	l := strings.ToLower(inf)
	if strings.Contains(l, "oppresive") {
		return false
	}
	if strings.HasSuffix(l, "_knockback") || strings.Contains(l, "silencer_last_word_disarm") {
		return false
	}
	if strings.Contains(strings.ReplaceAll(l, "_silencer", ""), "_silence") {
		return true
	}
	for _, s := range silenceSubstrings {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

func classifyHeal(inf string) bool {
	l := strings.ToLower(inf)
	return l == "modifier_item_urn_heal" ||
		l == "modifier_item_spirit_vessel_heal" ||
		l == "modifier_item_essence_distiller_heal" ||
		l == "modifier_bottle_regeneration" ||
		l == "modifier_item_polliwog_charm_buff"
}

func classifyDisarm(inf string) bool {
	l := strings.ToLower(inf)
	if strings.Contains(l, "lucky_shot") ||
		strings.Contains(l, "last_word") ||
		strings.Contains(l, "inner_fire") ||
		strings.Contains(l, "scatterblast") {
		return false
	}
	return strings.Contains(l, "disarm") ||
		strings.Contains(l, "heavens_halberd") ||
		strings.Contains(l, "concussive_grenade") ||
		strings.Contains(l, "fates_edict")
}

// trackedBuff reports whether a buff category is reported in buff_sources.
var TrackedBuff = map[string]bool{
	"save":           true,
	"purge":          true,
	"shield":         true,
	"spell_immunity": true,
	"invisibility":   true,
	"buff_stats":     true,
	"buff_haste":     true,
	"mana_restore":   true,
}

func HeroKey(s string) string {
	return strings.ReplaceAll(s, "_", "")
}

var camelRe = regexp.MustCompile(`([a-z])([A-Z])`)

func UnitToHeroName(unit string) string {
	const prefix = "CDOTA_Unit_Hero_"
	if !strings.HasPrefix(unit, prefix) {
		return unit
	}
	name := unit[len(prefix):]
	name = camelRe.ReplaceAllString(name, "${1}_${2}")
	return "npc_dota_hero_" + strings.ToLower(name)
}

func HeroID(name string) int {
	if id, ok := HeroMap[name]; ok {
		return id
	}
	return heroIDByKey[HeroKey(name)]
}

func IsBuilding(name string) bool {
	return strings.Contains(name, "tower") ||
		strings.Contains(name, "rax") ||
		strings.Contains(name, "fortress") ||
		strings.Contains(name, "building") ||
		strings.Contains(name, "shrine") ||
		strings.Contains(name, "effigy")
}
