package replay_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"context"

	"github.com/drawiks/manta-cli/internal/app"
)

// replayDir is the external corpus used for the parity harness. The large
// .dem files are not committed; set MANTA_REPLAY_DIR to point at them.
func replayDir() string {
	if d := os.Getenv("MANTA_REPLAY_DIR"); d != "" {
		return d
	}
	return "../../../replays"
}

func replayFile(id string) (string, bool) {
	dir := replayDir()
	candidates := []string{
		filepath.Join(dir, id+".dem"),
		filepath.Join(dir, id+".dem.bz2"),
		filepath.Join(dir, id+".dem.zst"),
		filepath.Join(dir, "..", id+".dem"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.Size() > 0 {
			return p, true
		}
	}
	return "", false
}

func TestParityWithOdotaCLI(t *testing.T) {
	dir := replayDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("replay corpus %s not present: %v", dir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".out.json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".out.json")
		dem, ok := replayFile(id)
		if !ok {
			t.Logf("%s: replay not present, skipping", id)
			continue
		}
		got, err := app.ParseFile(context.Background(), dem)
		if err != nil {
			t.Fatalf("%s: parse: %v", id, err)
		}
		gotJSON, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("%s: marshal: %v", id, err)
		}
		refJSON, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: read reference: %v", id, err)
		}
		var gotMap, refMap any
		if err := json.Unmarshal(gotJSON, &gotMap); err != nil {
			t.Fatalf("%s: decode got: %v", id, err)
		}
		if err := json.Unmarshal(refJSON, &refMap); err != nil {
			t.Fatalf("%s: decode ref: %v", id, err)
		}
		gotFlat := map[string]any{}
		refFlat := map[string]any{}
		flatten(gotMap, "", gotFlat)
		flatten(refMap, "", refFlat)
		for k, want := range refFlat {
			have, ok := gotFlat[k]
			if !ok {
				t.Errorf("%s: missing field %s", id, k)
				continue
			}
			if !closeEnough(k, want, have) {
				t.Errorf("%s: %s = %v, want %v", id, k, have, want)
			}
		}
		for k := range gotFlat {
			if _, ok := refFlat[k]; !ok {
				t.Errorf("%s: unexpected field %s", id, k)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Skip("no replay/reference pairs found")
	}
}

func flatten(v any, prefix string, out map[string]any) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flatten(val, key, out)
		}
	case []any:
		for i, val := range x {
			flatten(val, prefix+"["+itoa(i)+"]", out)
		}
	default:
		out[prefix] = v
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

// closeEnough encodes the known sampling residuals of the manta decoder:
// the final interval can be one tick behind clarity (networth), and death
// transitions are quantized to whole seconds (time_dead).
func closeEnough(key string, want, have any) bool {
	wf, wok := want.(float64)
	hf, hok := have.(float64)
	if !wok || !hok {
		return want == have
	}
	switch {
	case strings.HasSuffix(key, ".time_dead"):
		return math.Abs(wf-hf) <= 2
	case strings.HasSuffix(key, ".networth"):
		return math.Abs(wf-hf) <= 1
	case strings.HasSuffix(key, ".gpm"):
		return math.Abs(wf-hf) <= 1
	}
	return wf == hf
}

func TestJSONRoundtrip(t *testing.T) {
	dir := replayDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("replay corpus %s not present: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".out.json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".out.json")
		dem, ok := replayFile(id)
		if !ok {
			continue
		}
		res, err := app.ParseFile(context.Background(), dem)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if _, err := json.Marshal(res); err != nil {
			t.Errorf("%s: marshal: %v", id, err)
		}
	}
}
