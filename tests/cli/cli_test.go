package cli_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/drawiks/manta-cli/internal/cli"
	"github.com/drawiks/manta-cli/stats"
)

func run(args []string, stdin *bytes.Reader) (int, string, string) {
	var out, errBuf bytes.Buffer
	code := cli.Run(context.Background(), args, stdin, &out, &errBuf)
	return code, out.String(), errBuf.String()
}

func TestRunNoArgs(t *testing.T) {
	code, out, errBuf := run(nil, bytes.NewReader(nil))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf, "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf)
	}
	if out != "" {
		t.Errorf("stdout should be empty, got: %q", out)
	}
}

func TestRunBadFlag(t *testing.T) {
	code, _, _ := run([]string{"--nope", "x.dem"}, bytes.NewReader(nil))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunMissingFile(t *testing.T) {
	code, out, errBuf := run([]string{"does_not_exist.dem"}, bytes.NewReader(nil))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf, "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf)
	}
	if out != "" {
		t.Errorf("stdout should be empty on failure, got: %q", out)
	}
}

func TestRunHelpFlag(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		code, out, errBuf := run([]string{arg}, bytes.NewReader(nil))
		if code != 0 {
			t.Fatalf("%s: exit code = %d, want 0", arg, code)
		}
		if out != "" {
			t.Errorf("%s: stdout should be empty, got: %q", arg, out)
		}
		if !strings.Contains(errBuf, "usage:") {
			t.Errorf("%s: stderr should contain usage, got: %q", arg, errBuf)
		}
	}
}

func TestRunVersion(t *testing.T) {
	code, out, errBuf := run([]string{"--version"}, bytes.NewReader(nil))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "manta-cli") {
		t.Errorf("stdout should name manta-cli, got: %q", out)
	}
	if errBuf != "" {
		t.Errorf("stderr should be empty, got: %q", errBuf)
	}
}

func TestRunDashPrefixedPath(t *testing.T) {
	code, out, _ := run([]string{"-12345.dem"}, bytes.NewReader(nil))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if out != "" {
		t.Errorf("stdout should be empty, got: %q", out)
	}
}

func TestRunDumpFieldsMissingFile(t *testing.T) {
	code, out, errBuf := run([]string{"--dump-fields", "does_not_exist.dem"}, bytes.NewReader(nil))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf, "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf)
	}
	if out != "" {
		t.Errorf("stdout should be empty on failure, got: %q", out)
	}
}

func TestPrintTable(t *testing.T) {
	res := &stats.Match{
		MatchID:     42,
		DurationSec: 100,
		RadiantWin:  true,
		Players: []stats.Player{
			{
				Team: "radiant", Hero: "pudge", Name: "n", Kills: 8, Deaths: 3,
				Assists: 12, Level: 25, LastHits: 230, Networth: 18900,
				GPM: 512, XPM: 640, Healing: 5200, HeroDamage: 18750,
				DamageTaken: 12400, TimeDead: 240.5, TowerDamage: 1800,
				StunDuration: 14.3, FirstBlood: true, RunePickups: 2,
				GoldSpentWards: 900, GoldSpentSmoke: 200, GoldSpentDust: 80,
			},
		},
	}
	var buf bytes.Buffer
	cli.PrintTable(&buf, res)
	s := buf.String()
	for _, want := range []string{
		"match_id=42 duration=100s radiant_win=true",
		"slot", "hero", "lvl", "fb",
		"pudge", "512", "640", "true", "900",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("PrintTable output should contain %q, got:\n%s", want, s)
		}
	}
}

func TestRunStdinNoData(t *testing.T) {
	code, out, errBuf := run([]string{"--stdin"}, bytes.NewReader(nil))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf, "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf)
	}
	if out != "" {
		t.Errorf("stdout should be empty on failure, got: %q", out)
	}
}

func TestRunStdinGarbage(t *testing.T) {
	garbage := bytes.NewReader(bytes.Repeat([]byte("garbage"), 4096))
	code, _, errBuf := run([]string{"--stdin"}, garbage)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(errBuf, "error:") {
		t.Errorf("stderr should start with error:, got: %q", errBuf)
	}
}

func TestRunStdinSingleDashFlag(t *testing.T) {
	code, _, _ := run([]string{"-stdin"}, bytes.NewReader(nil))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunStdinWithPath(t *testing.T) {
	code, _, errBuf := run([]string{"--stdin", "x.dem"}, bytes.NewReader(nil))
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errBuf, "error:") {
		t.Errorf("stderr should contain an error prefix, got: %q", errBuf)
	}
}

func TestRunStdinParsesReplay(t *testing.T) {
	f, err := os.Open("../replays/9064911131.dem")
	if err != nil {
		t.Skip("replay fixture not present")
	}
	defer func() { _ = f.Close() }()

	var out, errBuf bytes.Buffer
	if code := cli.Run(context.Background(), []string{"--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	s := out.String()
	if !strings.HasPrefix(s, "{") || !strings.Contains(s, `"match_id":`) || !strings.Contains(s, `"players"`) {
		t.Errorf("stdout should contain match JSON, got: %q", s)
	}
}

func TestRunStdinDumpFields(t *testing.T) {
	f, err := os.Open("../replays/9064911131.dem")
	if err != nil {
		t.Skip("replay fixture not present")
	}
	defer func() { _ = f.Close() }()

	var out, errBuf bytes.Buffer
	if code := cli.Run(context.Background(), []string{"--dump-fields", "--stdin"}, f, &out, &errBuf); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "CDOTA_DataDire") {
		t.Errorf("stderr should contain CDOTA_DataDire, got: %q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for --dump-fields, got: %q", out.String())
	}
}
