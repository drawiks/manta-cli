package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/drawiks/manta-cli/internal/app"
	"github.com/drawiks/manta-cli/internal/model"
	"github.com/drawiks/manta-cli/internal/replay"
	"github.com/drawiks/manta-cli/internal/version"
)

const UsageLine = "usage: manta-cli <replay> [--stdin] [--output PATH] [--verbose] [--quiet] [--compact] [--dump-fields] [--version]"

type options struct {
	stdin      bool
	verbose    bool
	quiet      bool
	compact    bool
	dumpFields bool
	showVer    bool
	help       bool
	output     string
	paths      []string
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := pflag.NewFlagSet("manta-cli", pflag.ContinueOnError)
	fs.SetInterspersed(true)
	fs.SetOutput(stderr)
	fs.Usage = func() { printUsage(stderr) }

	var o options
	fs.BoolVar(&o.stdin, "stdin", false, "read the replay from standard input")
	fs.BoolVar(&o.verbose, "verbose", false, "print a human-readable table to stderr")
	fs.BoolVarP(&o.quiet, "quiet", "q", false, "suppress non-error output on stderr")
	fs.BoolVar(&o.compact, "compact", false, "emit compact JSON")
	fs.BoolVar(&o.dumpFields, "dump-fields", false, "dump PlayerResource and team-data fields to stderr")
	fs.BoolVar(&o.showVer, "version", false, "print version and exit")
	fs.BoolVarP(&o.help, "help", "h", false, "print help and exit")
	fs.StringVarP(&o.output, "output", "o", "", "write JSON to this path instead of stdout")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			printUsage(stderr)
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		printUsage(stderr)
		return 2
	}
	if o.help {
		printUsage(stderr)
		return 0
	}
	if o.showVer {
		_, _ = fmt.Fprintln(stdout, version.String())
		return 0
	}

	o.paths = fs.Args()
	if o.stdin && len(o.paths) > 0 {
		_, _ = fmt.Fprintln(stderr, "error: --stdin takes no replay argument")
		printUsage(stderr)
		return 2
	}
	if !o.stdin && (len(o.paths) != 1 || o.paths[0] == "-") {
		if len(o.paths) == 1 && o.paths[0] == "-" {
			o.stdin = true
		} else {
			_, _ = fmt.Fprintln(stderr, "error: exactly one replay path is required")
			printUsage(stderr)
			return 2
		}
	}

	if o.dumpFields {
		if o.stdin {
			return replay.DumpStdin(stdin, stderr)
		}
		return replay.DumpEntityFields(o.paths[0], stderr)
	}

	var res *model.Result
	var err error
	if o.stdin {
		res, err = app.ParseStream(ctx, stdin)
	} else {
		res, err = app.ParseFile(ctx, o.paths[0])
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 130
		}
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return emit(res, o, stdout, stderr)
}

func emit(res *model.Result, o options, stdout, stderr io.Writer) int {
	var out []byte
	var err error
	if o.compact {
		out, err = json.Marshal(res)
	} else {
		out, err = json.MarshalIndent(res, "", "  ")
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: encode: %v\n", err)
		return 1
	}
	out = append(out, '\n')

	if o.output != "" && o.output != "-" {
		if err := writeFileAtomic(o.output, out); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: write %s: %v\n", o.output, err)
			return 1
		}
	} else {
		if _, err := stdout.Write(out); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: write stdout: %v\n", err)
			return 1
		}
	}

	if o.verbose && !o.quiet {
		PrintTable(stderr, res)
	}
	return 0
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".manta-cli-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, UsageLine)
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Parse a Dota 2 replay and write the match statistics as JSON.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Inputs: .dem, .dem.bz2, .dem.zst, or - / --stdin.")
	_, _ = fmt.Fprintln(w, "Exit codes: 0 success, 1 parse or I/O error, 2 usage, 130 interrupted.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  manta-cli match.dem > match.json")
	_, _ = fmt.Fprintln(w, "  manta-cli match.dem.bz2 --verbose")
	_, _ = fmt.Fprintln(w, "  cat match.dem | manta-cli --stdin --compact")
	_, _ = fmt.Fprintln(w, "  manta-cli match.dem --output match.json")
}

func PrintTable(w io.Writer, res *model.Result) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(tw, "match_id=%d duration=%.0fs radiant_win=%v\n",
		res.MatchID, res.DurationSec, res.RadiantWin)
	_, _ = fmt.Fprintln(tw, "slot\tteam\thero\tname\tk\td\ta\tlvl\tlh\tnetworth\tgpm\txpm\theal\thdmg\tdtaken\tdead\ttdmg\tstun\tfb\trunes\twards\tsmoke\tdust")
	for i := range res.Players {
		p := &res.Players[i]
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.0f\t%d\t%.1f\t%v\t%d\t%d\t%d\t%d\n",
			i, p.Team, p.Hero, p.Name, p.Kills, p.Deaths, p.Assists, p.Level,
			p.LastHits, p.Networth, p.GPM, p.XPM, p.Healing, p.HeroDamage,
			p.DamageTaken, p.TimeDead, p.TowerDamage, p.StunDuration,
			p.FirstBlood, p.RunePickups, p.GoldSpentWards, p.GoldSpentSmoke, p.GoldSpentDust)
	}
	_ = tw.Flush()
}
