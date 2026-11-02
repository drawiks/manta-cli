package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/drawiks/manta-cli/internal/stats"
)

const UsageLine = "usage: manta_cli <match.dem|match.dem.bz2|match.dem.zst> [--stdin] [--verbose] [--dump-fields]"

// Run parses arguments manually so flags may appear both before and after the
// positional file path, matching the documented CLI (§3). stdout stays
// reserved for JSON; everything else goes to stderr.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var (
		verbose    bool
		dumpFields bool
		stdinMode  bool
		paths      []string
	)
	for _, a := range args {
		switch a {
		case "-h", "--help":
			fmt.Fprintln(stderr, UsageLine)
			return 0
		case "-verbose", "--verbose":
			verbose = true
		case "-dump-fields", "--dump-fields":
			dumpFields = true
		case "-stdin", "--stdin":
			stdinMode = true
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(stderr, "error: unknown flag %q\n%s\n", a, UsageLine)
				return 2
			}
			paths = append(paths, a)
		}
	}

	if stdinMode {
		if len(paths) > 0 {
			fmt.Fprintln(stderr, "error: --stdin takes no file argument\n"+UsageLine)
			return 2
		}
		if dumpFields {
			return stats.DumpStdin(stdin, stderr)
		}
		res, err := stats.ParseStdin(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return emit(res, verbose, stdout, stderr)
	}

	if len(paths) != 1 {
		fmt.Fprintln(stderr, "error: "+UsageLine)
		return 2
	}
	path := paths[0]

	if dumpFields {
		return stats.DumpEntityFields(path, stderr)
	}

	res, err := stats.ParseReplay(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return emit(res, verbose, stdout, stderr)
}

func emit(res *stats.Result, verbose bool, stdout, stderr io.Writer) int {
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(out))

	if verbose {
		PrintTable(stderr, res)
	}
	return 0
}

// PrintTable writes a human-readable summary of res to w.
func PrintTable(w io.Writer, res *stats.Result) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "match_id=%d duration=%.0fs radiant_win=%v\n",
		res.MatchID, res.DurationSec, res.RadiantWin)
	fmt.Fprintln(tw, "slot\tteam\thero\tname\tk\td\ta\tlvl\tlh\tnetworth\tgpm\txpm\theal\thdmg\tdtaken\tdead\ttdmg\tstun\tfb\trunes\twards\tsmoke\tdust")
	for i := range res.Players {
		p := &res.Players[i]
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.0f\t%d\t%.1f\t%v\t%d\t%d\t%d\t%d\n",
			i, p.Team, p.Hero, p.Name, p.Kills, p.Deaths, p.Assists, p.Level,
			p.LastHits, p.Networth, p.GPM, p.XPM, p.Healing, p.HeroDamage,
			p.DamageTaken, p.TimeDead, p.TowerDamage, p.StunDuration,
			p.FirstBlood, p.RunePickups, p.GoldSpentWards, p.GoldSpentSmoke, p.GoldSpentDust)
	}
	tw.Flush()
}
