package stats

import "testing"

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
	players := []Player{
		{PlayerID: 0, Team: "radiant"},
		{PlayerID: 1, Team: "radiant"},
		{PlayerID: 2, Team: "radiant"},
		{PlayerID: 3, Team: "radiant"},
		{PlayerID: 4, Team: "radiant"},
	}
	samples := []*roleSamples{
		safeSamples(100),
		safeSamples(40),
		midSamples(80),
		offSamples(60),
		offSamples(20),
	}
	farm := []roleFarm{
		{networth: 20000, lh: 300, xp: 20000},
		{networth: 6000, lh: 30, xp: 8000},
		{networth: 18000, lh: 250, xp: 19000},
		{networth: 16000, lh: 200, xp: 17000},
		{networth: 5000, lh: 20, xp: 7000},
	}
	in := make([]roleInput, len(players))
	for i := range players {
		in[i] = roleInput{p: &players[i], stats: samples[i], farm: farm[i]}
	}
	assignRoles(in)
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
	p := Player{PlayerID: 0, Team: "radiant"}
	assignRoles([]roleInput{{p: &p}})
	if p.Position != 0 || p.Lane != "" {
		t.Errorf("no samples should leave role empty, got pos=%d lane=%q", p.Position, p.Lane)
	}
}

func safeSamples(n int) *roleSamples {
	s := &roleSamples{}
	for i := 0; i < n; i++ {
		s.sample(22000, 9000)
	}
	return s
}

func offSamples(n int) *roleSamples {
	s := &roleSamples{}
	for i := 0; i < n; i++ {
		s.sample(9000, 22000)
	}
	return s
}

func midSamples(n int) *roleSamples {
	s := &roleSamples{}
	for i := 0; i < n; i++ {
		s.sample(16000, 16000)
	}
	return s
}
