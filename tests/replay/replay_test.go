package replay_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"manta-cli/internal/stats"
)

const replayDir = "../replays"

func replayPath(name string) string {
	return filepath.Join(replayDir, name)
}

func parseReplay(t *testing.T, name string) *stats.Result {
	t.Helper()
	path := replayPath(name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("replay fixture %s not present", path)
	}
	res, err := stats.ParseReplay(path)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return res
}

func TestAllReplaysStunAudit(t *testing.T) {
	entries, err := os.ReadDir(replayDir)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".dem") || strings.HasSuffix(name, ".dem.bz2") || strings.HasSuffix(name, ".dem.zst") {
			files = append(files, filepath.Join(replayDir, name))
		}
	}
	if len(files) == 0 {
		t.Fatal("no replays in tests/replays/")
	}

	var issues []string
	for _, path := range files {
		fi, _ := os.Stat(path)
		if fi != nil && fi.Size() == 0 {
			issues = append(issues, fmt.Sprintf("%s: empty file", filepath.Base(path)))
			continue
		}

		res, err := stats.ParseReplay(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: PARSE ERROR: %v", filepath.Base(path), err))
			continue
		}

		t.Logf("\n======== %s match=%d dur=%ds radiant_win=%v ========",
			filepath.Base(path), res.MatchID, res.DurationSec, res.RadiantWin)

		for _, p := range res.Players {
			if p.StunDuration < 0 {
				issues = append(issues, fmt.Sprintf("%s %s: negative stun %.4f", filepath.Base(path), p.Hero, p.StunDuration))
			}
			if p.StunDuration > 500 {
				issues = append(issues, fmt.Sprintf("%s %s: suspiciously high stun %.1f", filepath.Base(path), p.Hero, p.StunDuration))
			}

			if p.StunDuration > 0 || len(p.StunSources) > 0 {
				t.Logf("  %-18s stun=%8.3f  sources=%d", p.Hero, p.StunDuration, len(p.StunSources))
				for _, s := range p.StunSources {
					t.Logf("      %7.3fs  %-10s  %s", s.Seconds, s.Category, s.Inflictor)
				}
			}
		}
	}

	if len(issues) > 0 {
		t.Log("\n======== ISSUES ========")
		for _, msg := range issues {
			t.Log("  " + msg)
		}
		t.Errorf("%d issue(s) found", len(issues))
	}
}

func TestAllReplaysJSONRoundtrip(t *testing.T) {
	entries, err := os.ReadDir(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(replayDir, e.Name())
		fi, _ := os.Stat(path)
		if fi != nil && fi.Size() == 0 {
			continue
		}
		res, err := stats.ParseReplay(path)
		if err != nil {
			continue
		}
		if _, err := json.Marshal(res); err != nil {
			t.Errorf("%s: json marshal: %v", e.Name(), err)
		}
	}
}

func TestAxeStunRegression(t *testing.T) {
	res := parseReplay(t, "8940258922.dem.bz2")
	for _, p := range res.Players {
		if p.Hero != "Axe" {
			continue
		}
		if p.StunDuration > 70 {
			t.Errorf("Axe stun_duration=%.3f, want ~scoreboard (<70)", p.StunDuration)
		}
		for _, s := range p.StunSources {
			if strings.Contains(s.Inflictor, "armor") {
				t.Errorf("unexpected self-buff source: %+v", s)
			}
		}
		return
	}
	t.Fatal("no Axe")
}

func TestVenomancerStunRegression(t *testing.T) {
	res := parseReplay(t, "8939160389.dem")
	for _, p := range res.Players {
		if p.Hero == "Venomancer" && p.StunDuration > 1 {
			t.Errorf("Venomancer stun_duration=%.3f, want 0 (gale not in m_fStuns)", p.StunDuration)
		}
	}
}

func TestNagaSongRegression(t *testing.T) {
	res := parseReplay(t, "8943360001.dem")
	for _, p := range res.Players {
		if p.Hero != "Naga_Siren" {
			continue
		}
		// m_fStuns is present and authoritative (matches the in-game scoreboard
		// / OpenDota on modern matches), so Naga's stun is the scoreboard total;
		// the paired ADD→REMOVE sleep credit is not layered on top.
		if p.StunDuration < 76 || p.StunDuration > 80 {
			t.Errorf("Naga stun_duration=%.3f, want ~77.7 (scoreboard m_fStuns)", p.StunDuration)
		}
		for _, s := range p.StunSources {
			if s.Category == "sleep" {
				t.Errorf("scoreboard matches must not carry sleep supplemental: %+v", s)
			}
			if s.Inflictor != "scoreboard" {
				t.Errorf("scoreboard stun source = %+v, want single scoreboard entry", s)
			}
		}
		return
	}
	t.Fatal("no Naga")
}

func TestAllReplaysBuffAudit(t *testing.T) {
	entries, err := os.ReadDir(replayDir)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".dem") || strings.HasSuffix(name, ".dem.bz2") || strings.HasSuffix(name, ".dem.zst") {
			files = append(files, filepath.Join(replayDir, name))
		}
	}
	if len(files) == 0 {
		t.Fatal("no replays in tests/replays/")
	}

	known := map[string]bool{
		"heal": true, "save": true, "purge": true, "shield": true,
		"spell_immunity": true, "invisibility": true, "buff_stats": true,
		"buff_haste": true, "mana_restore": true,
	}

	var issues []string
	for _, path := range files {
		fi, _ := os.Stat(path)
		if fi != nil && fi.Size() == 0 {
			continue
		}
		res, err := stats.ParseReplay(path)
		if err != nil {
			continue
		}
		for _, p := range res.Players {
			if len(p.BuffSources) == 0 {
				continue
			}
			t.Logf("  %-18s buffs=%d", p.Hero, len(p.BuffSources))
			for _, b := range p.BuffSources {
				t.Logf("      %9.2f  %3dx  %-14s %s", b.Value, b.Count, b.Category, b.Inflictor)
				if b.Value < 0 || b.Count < 0 {
					issues = append(issues, fmt.Sprintf("%s %s: negative buff %+v", filepath.Base(path), p.Hero, b))
				}
				if b.Inflictor == "" || !known[b.Category] {
					issues = append(issues, fmt.Sprintf("%s %s: bad buff source %+v", filepath.Base(path), p.Hero, b))
				}
			}
		}
	}

	if len(issues) > 0 {
		for _, msg := range issues {
			t.Log("  " + msg)
		}
		t.Errorf("%d issue(s) found", len(issues))
	}
}

func TestTreantHealRegression(t *testing.T) {
	res := parseReplay(t, "8926354517.dem")
	for _, p := range res.Players {
		if p.Hero != "Treant" {
			continue
		}
		var livingArmor float64
		for _, b := range p.BuffSources {
			if b.Inflictor == "treant_living_armor" {
				livingArmor = b.Value
			}
		}
		if livingArmor < 500 {
			t.Errorf("Treant living armor heal=%.1f, want >500 (full-ally regen)", livingArmor)
		}
		return
	}
	t.Fatal("no Treant")
}

func TestCentaurBuffRegression(t *testing.T) {
	res := parseReplay(t, "8940258922.dem.bz2")
	for _, p := range res.Players {
		if p.Hero != "Centaur" {
			continue
		}
		var pipe, stampede float64
		for _, b := range p.BuffSources {
			switch b.Inflictor {
			case "item_pipe_of_insight":
				pipe = b.Value
			case "centaur_stampede":
				stampede = b.Value
			}
		}
		if pipe < 200 {
			t.Errorf("Centaur pipe shield=%.1f, want >200s", pipe)
		}
		if stampede < 100 {
			t.Errorf("Centaur stampede haste=%.1f, want >100s", stampede)
		}
		return
	}
	t.Fatal("no Centaur")
}

func TestSelfCastBuffRegression(t *testing.T) {
	// Hurricane Pike self-casts are the cleanest self-cast signal: in this
	// fixture every pike-active modifier lands on its owner (attacker==target),
	// so before the self-cast relaxation these buffs were never credited.
	res := parseReplay(t, "8930714427_1322621597.dem.bz2")
	for _, p := range res.Players {
		if p.Hero != "Windrunner" {
			continue
		}
		for _, b := range p.BuffSources {
			if b.Inflictor == "item_hurricane_pike" {
				if b.Count < 5 {
					t.Errorf("Windrunner pike count=%d, want >=5 (self-casts)", b.Count)
				}
				if b.Value <= 0 {
					t.Errorf("Windrunner pike value=%.2f, want >0", b.Value)
				}
				return
			}
		}
		t.Fatal("no hurricane pike buff on Windrunner")
	}
	t.Fatal("no Windrunner")
}

func TestNightStalkerStunRegression(t *testing.T) {
	res := parseReplay(t, "8930714427_1322621597.dem.bz2")
	var ns *stats.Player
	for i := range res.Players {
		if res.Players[i].Hero == "NightStalker" {
			ns = &res.Players[i]
			break
		}
	}
	if ns == nil {
		t.Fatal("no NightStalker")
	}
	// The scoreboard m_fStuns section is present and authoritative (matches the
	// in-game scoreboard / OpenDota): crippling fear silences are not in
	// m_fStuns, so NightStalker's stun is the ~1.97s of hard disable the
	// scoreboard reports, not the paired-silence uptime. The supplemental model
	// is only a code-level fallback for replays where the scoreboard section is
	// missing entirely (none of the current fixtures exercise it).
	if ns.StunDuration < 1.9 || ns.StunDuration > 2.1 {
		t.Errorf("NightStalker stun_duration=%.3f, want ~1.97 (scoreboard m_fStuns)", ns.StunDuration)
	}
	for _, s := range ns.StunSources {
		if s.Category == "fear" {
			t.Errorf("crippling fear must not be fear: %+v", s)
		}
	}
}

// TestNewMatchesExactFields locks KDA / last_hits / level against the OpenDota
// /matches API values (70/70 exact at authoring time) for the modern-patch
// fixtures, and pins first_blood attribution. Fixtures are not in git: absent
// fixtures skip (parseReplay).
func TestNewMatchesExactFields(t *testing.T) {
	type want struct {
		hero     string
		kills    int32
		deaths   int32
		assists  int32
		lastHits uint32
		level    int32
	}
	cases := []struct {
		file  string
		first string // hero credited with first blood (combat-log model; OD exposes firstblood_claimed only on some matches)
		wants []want
	}{
		{"8940493468.dem.bz2", "Techies", []want{
			{"Sniper", 3, 9, 10, 113, 17}, {"Pugna", 4, 10, 8, 161, 21},
			{"SpiritBreaker", 1, 18, 11, 88, 18}, {"Nevermore", 14, 9, 4, 334, 25},
			{"Alchemist", 0, 11, 6, 282, 19}, {"Lina", 13, 2, 17, 375, 27},
			{"Shadow_Demon", 10, 3, 25, 71, 23}, {"Techies", 15, 6, 14, 154, 23},
			{"Luna", 11, 3, 10, 498, 26}, {"DoomBringer", 8, 8, 17, 283, 25},
		}},
		{"8941955514.dem.bz2", "PhantomLancer", []want{
			{"Axe", 10, 8, 10, 200, 20}, {"PhantomLancer", 6, 8, 9, 400, 23},
			{"Enchantress", 3, 14, 12, 82, 18}, {"Pangolier", 3, 12, 10, 300, 21},
			{"Ogre_Magi", 2, 14, 13, 110, 19}, {"NightStalker", 6, 7, 27, 200, 22},
			{"Mirana", 13, 5, 31, 181, 24}, {"Juggernaut", 16, 5, 10, 516, 27},
			{"EmberSpirit", 10, 3, 23, 201, 22}, {"Shadow_Demon", 9, 5, 38, 111, 22},
		}},
		{"8941982907.dem.bz2", "Omniknight", []want{
			{"Omniknight", 10, 1, 16, 244, 23}, {"Kez", 5, 6, 12, 448, 24},
			{"Viper", 9, 2, 8, 419, 25}, {"Magnataur", 5, 10, 23, 21, 20},
			{"Hoodwink", 6, 6, 23, 57, 21}, {"Clinkz", 10, 5, 3, 397, 23},
			{"Disruptor", 2, 9, 10, 68, 17}, {"Lina", 7, 2, 4, 440, 25},
			{"BountyHunter", 4, 15, 12, 46, 17}, {"NightStalker", 2, 5, 9, 349, 22},
		}},
		{"8943200760.dem.bz2", "AntiMage", []want{
			{"CrystalMaiden", 5, 10, 19, 158, 24}, {"SandKing", 15, 3, 17, 373, 27},
			{"Lich", 3, 5, 22, 86, 21}, {"StormSpirit", 11, 1, 11, 378, 28},
			{"DrowRanger", 9, 5, 10, 479, 27}, {"Lion", 5, 11, 10, 97, 22},
			{"AntiMage", 9, 6, 3, 507, 26}, {"Terrorblade", 1, 9, 7, 271, 22},
			{"Tinker", 6, 8, 8, 378, 26}, {"Pudge", 3, 9, 17, 126, 23},
		}},
		{"8943250018.dem.bz2", "Obsidian_Destroyer", []want{
			{"SkeletonKing", 21, 6, 10, 329, 27}, {"Zuus", 6, 15, 29, 115, 20},
			{"Snapfire", 8, 11, 24, 234, 27}, {"Undying", 8, 19, 22, 112, 22},
			{"Kez", 11, 12, 12, 309, 24}, {"Lina", 7, 12, 28, 358, 27},
			{"AntiMage", 24, 5, 12, 540, 30}, {"Lion", 6, 15, 20, 101, 23},
			{"Obsidian_Destroyer", 18, 11, 17, 227, 26}, {"Tidehunter", 6, 11, 33, 393, 27},
		}},
		{"8943313464.dem.bz2", "Necrolyte", []want{
			{"Invoker", 8, 6, 8, 142, 20}, {"Lion", 1, 9, 11, 12, 15},
			{"TemplarAssassin", 13, 4, 11, 364, 23}, {"Largo", 4, 5, 14, 233, 22},
			{"Terrorblade", 5, 9, 15, 82, 18}, {"Phoenix", 8, 5, 7, 169, 18},
			{"SpiritBreaker", 3, 8, 12, 31, 13}, {"NightStalker", 1, 5, 8, 153, 16},
			{"Techies", 9, 10, 10, 47, 14}, {"Necrolyte", 10, 4, 7, 342, 22},
		}},
		{"8943366462.dem.bz2", "Dawnbreaker", []want{
			{"Kez", 10, 4, 23, 196, 17}, {"Huskar", 17, 5, 8, 84, 16},
			{"Winter_Wyvern", 2, 3, 32, 20, 17}, {"Magnataur", 3, 1, 21, 70, 16},
			{"Lion", 8, 4, 19, 45, 16}, {"Dawnbreaker", 7, 6, 4, 101, 16},
			{"Snapfire", 4, 6, 3, 149, 16}, {"Earthshaker", 0, 10, 8, 30, 10},
			{"Leshrac", 6, 9, 6, 144, 14}, {"Wisp", 0, 10, 10, 5, 13},
		}},
	}

	for _, c := range cases {
		res := parseReplay(t, c.file)
		byHero := make(map[string]*stats.Player, len(res.Players))
		for i := range res.Players {
			byHero[res.Players[i].Hero] = &res.Players[i]
		}
		if len(byHero) != len(c.wants) {
			t.Errorf("%s: %d players, want %d", c.file, len(byHero), len(c.wants))
		}
		for _, w := range c.wants {
			p, ok := byHero[w.hero]
			if !ok {
				t.Errorf("%s: no player %s", c.file, w.hero)
				continue
			}
			if p.Kills != w.kills || p.Deaths != w.deaths || p.Assists != w.assists ||
				p.LastHits != w.lastHits || p.Level != w.level {
				t.Errorf("%s %s: got %d/%d/%d lh=%d lvl=%d, want %d/%d/%d lh=%d lvl=%d",
					c.file, w.hero, p.Kills, p.Deaths, p.Assists, p.LastHits, p.Level,
					w.kills, w.deaths, w.assists, w.lastHits, w.level)
			}
			if p.FirstBlood != (w.hero == c.first) {
				t.Errorf("%s %s: first_blood=%v, want %v", c.file, w.hero, p.FirstBlood, w.hero == c.first)
			}
		}
	}
}

// TestNewMatchesBuffRegression pins buffs_duration on a few notable players of
// the modern-patch fixtures so a silent regression in the paired-uptime buff
// model (double counting, dropped category) is caught even though the values
// are not externally validated.
func TestNewMatchesBuffRegression(t *testing.T) {
	type want struct {
		hero    string
		buffMin float64
		buffMax float64
	}
	cases := []struct {
		file  string
		wants []want
	}{
		{"8940493468.dem.bz2", []want{
			{"Pugna", 76, 79},
			{"Shadow_Demon", 145, 149},
		}},
		{"8941955514.dem.bz2", []want{
			{"Enchantress", 150, 154},
			{"Ogre_Magi", 2010, 2016},
		}},
		{"8943366462.dem.bz2", []want{
			{"Magnataur", 598, 601},
			{"Winter_Wyvern", 81, 83},
			{"Wisp", 1, 3},
		}},
	}
	for _, c := range cases {
		res := parseReplay(t, c.file)
		byHero := make(map[string]*stats.Player, len(res.Players))
		for i := range res.Players {
			byHero[res.Players[i].Hero] = &res.Players[i]
		}
		for _, w := range c.wants {
			p, ok := byHero[w.hero]
			if !ok {
				t.Errorf("%s: no player %s", c.file, w.hero)
				continue
			}
			if p.BuffsDuration < w.buffMin || p.BuffsDuration > w.buffMax {
				t.Errorf("%s %s: buffs_duration=%.2f, want [%.0f, %.0f]", c.file, w.hero, p.BuffsDuration, w.buffMin, w.buffMax)
			}
		}
	}
}
