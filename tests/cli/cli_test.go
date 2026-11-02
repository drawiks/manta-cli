package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/drawiks/manta-cli/internal/cli"
	"github.com/drawiks/manta-cli/internal/stats"
)

func TestRunNoArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run(nil, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--nope", "x.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"does_not_exist.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
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
	if code := cli.Run([]string{"-h"}, bytes.NewReader(nil), &out, &errBuf); code != 0 {
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
	if code := cli.Run([]string{"--help"}, bytes.NewReader(nil), &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(errBuf.String(), "usage:") {
		t.Errorf("stderr should contain the usage line, got: %q", errBuf.String())
	}
}

func TestRunDashPrefixedPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"-12345.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty, got: %q", out.String())
	}
}

func TestRunDumpFieldsMissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--dump-fields", "does_not_exist.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
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
	res := &stats.Result{
		MatchID:     42,
		DurationSec: 100,
		RadiantWin:  true,
		Players: []stats.Player{
			{Team: "radiant", Hero: "Pudge", Name: "n", Kills: 8, Deaths: 3,
				Assists: 12, Level: 25, LastHits: 230, Networth: 18900,
				GPM: 512, XPM: 640, Healing: 5200, HeroDamage: 18750,
				DamageTaken: 12400, TimeDead: 240.5, TowerDamage: 1800,
				StunDuration: 14.3, FirstBlood: true, RunePickups: 2,
				GoldSpentWards: 900, GoldSpentSmoke: 200, GoldSpentDust: 80},
		},
	}
	var buf bytes.Buffer
	cli.PrintTable(&buf, res)
	s := buf.String()
	for _, want := range []string{
		"match_id=42 duration=100s radiant_win=true",
		"slot", "hero", "lvl", "fb",
		"Pudge", "512", "640", "true", "900",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("PrintTable output should contain %q, got:\n%s", want, s)
		}
	}
}

func TestRunStdinNoData(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--stdin"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf.String(), "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty on failure, got: %q", out.String())
	}
}

func TestRunStdinGarbage(t *testing.T) {
	var out, errBuf bytes.Buffer
	garbage := bytes.NewReader(bytes.Repeat([]byte("garbage"), 4096))
	if code := cli.Run([]string{"--stdin"}, garbage, &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf.String(), "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf.String())
	}
}

func TestRunStdinSingleDashFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"-stdin"}, bytes.NewReader(nil), &out, &errBuf); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestRunStdinWithPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--stdin", "x.dem"}, bytes.NewReader(nil), &out, &errBuf); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf.String())
	}
}

func TestRunStdinParsesReplay(t *testing.T) {
	f, err := os.Open("../replays/8926354517.dem")
	if err != nil {
		t.Skip("replay fixture tests/replays/8926354517.dem not present")
	}
	defer f.Close()

	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	s := out.String()
	if !strings.HasPrefix(s, "{") {
		t.Errorf("stdout should be JSON, got: %q", s)
	}
	if !strings.Contains(s, `"match_id":`) {
		t.Errorf("stdout should contain match_id, got: %q", s)
	}
	if !strings.Contains(s, `"players"`) {
		t.Errorf("stdout should contain players, got: %q", s)
	}
}

func TestRunStdinDumpFields(t *testing.T) {
	f, err := os.Open("../replays/8926354517.dem")
	if err != nil {
		t.Skip("replay fixture tests/replays/8926354517.dem not present")
	}
	defer f.Close()

	var out, errBuf bytes.Buffer
	if code := cli.Run([]string{"--dump-fields", "--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "CDOTA_DataDire") {
		t.Errorf("stderr should contain CDOTA_DataDire, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for --dump-fields, got: %q", out.String())
	}
}
