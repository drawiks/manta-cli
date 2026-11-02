package roles

import (
	"testing"

	"github.com/drawiks/manta-cli/internal/model"
)

func TestClassifyZoneAndLane(t *testing.T) {
	if classifyZone(16000, 16000) != zoneMid {
		t.Error("map diagonal should be mid")
	}
	if classifyZone(22000, 8000) != zoneSafe {
		t.Error("bottom-right should be safe zone")
	}
	if classifyZone(8000, 22000) != zoneOff {
		t.Error("top-left should be off zone")
	}
	if classifyZone(8000, 8000) != zoneOther {
		t.Error("radiant base should be other")
	}
	if laneOf(zoneSafe, "radiant") != laneSafe || laneOf(zoneSafe, "dire") != laneOff {
		t.Error("safe zone must mirror by team")
	}
	want := map[int]string{1: "safe", 2: "mid", 3: "off", 4: "supp", 5: "hard_supp", 0: ""}
	for pos, lane := range want {
		if got := laneName(pos); got != lane {
			t.Errorf("laneName(%d) = %q, want %q", pos, got, lane)
		}
	}
}

func TestAssignRoles(t *testing.T) {
	players := []model.Player{
		{PlayerID: 0, Team: "radiant"},
		{PlayerID: 1, Team: "radiant"},
		{PlayerID: 2, Team: "radiant"},
		{PlayerID: 3, Team: "radiant"},
		{PlayerID: 4, Team: "radiant"},
	}
	samples := []*Samples{
		safeSamples(100),
		safeSamples(40),
		midSamples(80),
		offSamples(60),
		offSamples(20),
	}
	farm := []Farm{
		{Networth: 20000, LH: 300, XP: 20000},
		{Networth: 6000, LH: 30, XP: 8000},
		{Networth: 18000, LH: 250, XP: 19000},
		{Networth: 16000, LH: 200, XP: 17000},
		{Networth: 5000, LH: 20, XP: 7000},
	}
	in := make([]Input, len(players))
	for i := range players {
		in[i] = Input{P: &players[i], Stats: samples[i], Farm: farm[i]}
	}
	AssignRoles(in)
	want := []struct {
		pos  int
		lane string
	}{{1, "safe"}, {5, "hard_supp"}, {2, "mid"}, {3, "off"}, {4, "supp"}}
	for i, w := range want {
		if players[i].Position != w.pos || players[i].Lane != w.lane {
			t.Errorf("player %d: got pos=%d lane=%q, want pos=%d lane=%q",
				i, players[i].Position, players[i].Lane, w.pos, w.lane)
		}
	}
}

func TestAssignRolesNoSamples(t *testing.T) {
	p := model.Player{PlayerID: 0, Team: "radiant"}
	AssignRoles([]Input{{P: &p}})
	if p.Position != 0 || p.Lane != "" {
		t.Errorf("no samples should leave role empty, got pos=%d lane=%q", p.Position, p.Lane)
	}
}

func safeSamples(n int) *Samples {
	s := &Samples{}
	for i := 0; i < n; i++ {
		s.Sample(22000, 9000)
	}
	return s
}

func offSamples(n int) *Samples {
	s := &Samples{}
	for i := 0; i < n; i++ {
		s.Sample(9000, 22000)
	}
	return s
}

func midSamples(n int) *Samples {
	s := &Samples{}
	for i := 0; i < n; i++ {
		s.Sample(16000, 16000)
	}
	return s
}
