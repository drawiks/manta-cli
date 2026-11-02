package replay

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dotabuff/manta"
)

// The dump goes to stderr so stdout remains the JSON contract.
func DumpEntityFields(path string, w io.Writer) int {
	r, closer, err := OpenReader(path)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error: %v\n", err)
		return 1
	}
	return dumpStream(r, closer, w)
}

func DumpStdin(r io.Reader, w io.Writer) int {
	reader, closer, err := decompressReader(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error: %v\n", err)
		return 1
	}
	return dumpStream(reader, closer, w)
}

func dumpStream(r io.Reader, closer io.Closer, w io.Writer) int {
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	p, err := manta.NewStreamParser(r)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error: %v\n", err)
		return 1
	}

	var pr *manta.Entity
	teams := make(map[string]*manta.Entity)
	heroes := make(map[string]*manta.Entity)
	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		if !op.Flag(manta.EntityOpEntered) {
			return nil
		}
		switch {
		case e.GetClassName() == prClass:
			pr = e
		case e.GetClassName() == dataTeamRadiantClass || e.GetClassName() == dataTeamDireClass:
			teams[e.GetClassName()] = e
		case strings.HasPrefix(e.GetClassName(), heroClassPrefix):
			heroes[e.GetClassName()] = e
		}
		return nil
	})

	if err := p.Start(); err != nil {
		_, _ = fmt.Fprintf(w, "error: %v\n", err)
		return 1
	}

	if pr == nil {
		_, _ = fmt.Fprintln(w, "error: CDOTA_PlayerResource not found")
		return 1
	}

	dumpSortedMap(w, pr.Map())
	for _, cls := range []string{dataTeamRadiantClass, dataTeamDireClass} {
		_, _ = fmt.Fprintf(w, "\n[%s]\n", cls)
		if e, ok := teams[cls]; ok {
			dumpSortedMap(w, e.Map())
		} else {
			_, _ = fmt.Fprintln(w, "(not found)")
		}
	}

	_, _ = fmt.Fprintln(w, "\n[heroes]")
	hk := make([]string, 0, len(heroes))
	for c := range heroes {
		hk = append(hk, c)
	}
	sort.Strings(hk)
	for _, c := range hk {
		e := heroes[c]
		pid := playerIDOf(e)
		team := 0
		if v, ok := e.GetInt32("m_iTeamNum"); ok {
			team = int(v)
		} else if v, ok := e.GetUint32("m_iTeamNum"); ok {
			team = int(v)
		}
		_, _ = fmt.Fprintf(w, "%s  handle=0x%x  m_iPlayerID=%d  m_iTeamNum=%d\n",
			c, handleOf(e), pid, team)
	}

	if len(hk) > 0 {
		_, _ = fmt.Fprintf(w, "\n[hero map: %s]\n", hk[0])
		for _, k := range []string{"m_iPlayerID", "m_iEconSpecifyingPlayerID", "m_nPlayerOwnerID", "m_iIsControllableByPlayer64"} {
			_, _ = fmt.Fprintf(w, "%s [map type=%T]\n", k, heroes[hk[0]].Map()[k])
		}
		dumpSortedMap(w, heroes[hk[0]].Map())
	}
	return 0
}

func dumpSortedMap(w io.Writer, m map[string]interface{}) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%s = %v\n", k, m[k])
	}
}
