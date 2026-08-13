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
		if p.StunDuration < 120 || p.StunDuration > 135 {
			t.Errorf("Naga stun_duration=%.3f, want ~127 (sb + enemy song gap)", p.StunDuration)
		}
		var sleep float64
		for _, s := range p.StunSources {
			if s.Category == "sleep" {
				sleep += s.Seconds
			}
		}
		if sleep < 45 || sleep > 55 {
			t.Errorf("Naga sleep supplemental=%.3f, want ~49 (enemies only)", sleep)
		}
		return
	}
	t.Fatal("no Naga")
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
	if ns.StunDuration < 70 || ns.StunDuration > 95 {
		t.Errorf("NightStalker stun_duration=%.3f, want ~84 (sb + paired silence)", ns.StunDuration)
	}
	for _, s := range ns.StunSources {
		if s.Category == "fear" {
			t.Errorf("crippling fear must not be fear: %+v", s)
		}
		if s.Inflictor == "night_stalker_crippling_fear" && s.Category != "silence" {
			t.Errorf("crippling fear want silence: %+v", s)
		}
	}
}