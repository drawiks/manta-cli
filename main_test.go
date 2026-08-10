package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunNoArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run(nil, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"--nope", "x.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"does_not_exist.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf.String(), "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on failure, got: %q", out.String())
	}
}

func TestRunHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"-h"}, bytes.NewReader(nil), &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for -h, got: %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "usage:") {
		t.Errorf("stderr should contain the usage line, got: %q", errBuf.String())
	}
}

func TestRunDoubleDashHelp(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"--help"}, bytes.NewReader(nil), &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(errBuf.String(), "usage:") {
		t.Errorf("stderr should contain the usage line, got: %q", errBuf.String())
	}
}

// TestRunDashPrefixedPath checks that a replay file whose name starts with "-"
// is rejected by the flag parser (documented limitation), not parsed.
func TestRunDashPrefixedPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"-12345.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty, got: %q", out.String())
	}
}

func TestRunDumpFieldsMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"--dump-fields", "does_not_exist.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on failure, got: %q", out.String())
	}
}

func TestPrintTable(t *testing.T) {
	res := &Result{
		MatchID:     42,
		DurationSec: 100,
		RadiantWin:  true,
		Players: []Player{
			{Team: "radiant", Hero: "Pudge", Name: "n", Kills: 8, Deaths: 3,
				Assists: 12, Level: 25, LastHits: 230, Networth: 18900,
				GPM: 512, XPM: 640, Healing: 5200, HeroDamage: 18750,
				DamageTaken: 12400, TimeDead: 240.5, TowerDamage: 1800,
				StunDuration: 14.3, FirstBlood: true, RunePickups: 2,
				GoldWards: 900, GoldSmoke: 200, GoldDust: 80},
		},
	}
	var buf bytes.Buffer
	printTable(&buf, res)
	s := buf.String()
	for _, want := range []string{
		"match_id=42 duration=100s radiant_win=true",
		"slot", "hero", "lvl", "fb",
		"Pudge", "512", "640", "true", "900",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("printTable output should contain %q, got:\n%s", want, s)
		}
	}
}
