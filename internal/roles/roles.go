package roles

import (
	"sort"

	"github.com/drawiks/manta-cli/internal/model"
)

// Lane and position detection uses the first WindowSec seconds of hero
// world positions plus the farm profile at the end of that window.
const (
	WindowSec = 600.0

	midBand = 2000.0
	midXMin = 10500.0
	midXMax = 22000.0
	safeRY  = 12500.0
	safeRX  = 20000.0
	offRX   = 12500.0
	offRY   = 19000.0

	zoneDominanceFrac = 0.45
)

type roleZone int

const (
	zoneOther roleZone = iota
	zoneMid
	zoneSafe
	zoneOff
)

func classifyZone(x, y float64) roleZone {
	if x > midXMin-midBand && x < midXMax+midBand && y > midXMin-midBand && y < midXMax+midBand {
		if dx := x - y; dx > -midBand && dx < midBand {
			return zoneMid
		}
	}
	if y < safeRY && x > safeRX {
		return zoneSafe
	}
	if x < offRX && y > offRY {
		return zoneOff
	}
	return zoneOther
}

type laneKind int

const (
	laneUnknown laneKind = iota
	laneMid
	laneSafe
	laneOff
)

type Samples struct {
	zones [4]int
	total int
}

func (r *Samples) Sample(x, y float64) {
	r.zones[classifyZone(x, y)]++
	r.total++
}

func (r *Samples) dominant() (roleZone, bool) {
	if r.total == 0 {
		return zoneOther, false
	}
	best, bestN := zoneOther, 0
	for z := zoneMid; z <= zoneOff; z++ {
		if r.zones[z] > bestN {
			best, bestN = z, r.zones[z]
		}
	}
	return best, float64(bestN) >= zoneDominanceFrac*float64(r.total)
}

func laneOf(z roleZone, team string) laneKind {
	switch z {
	case zoneMid:
		return laneMid
	case zoneSafe:
		if team == "dire" {
			return laneOff
		}
		return laneSafe
	case zoneOff:
		if team == "dire" {
			return laneSafe
		}
		return laneOff
	}
	return laneUnknown
}

func laneName(pos int) string {
	switch pos {
	case 1:
		return "safe"
	case 2:
		return "mid"
	case 3:
		return "off"
	case 4:
		return "supp"
	case 5:
		return "hard_supp"
	}
	return ""
}

type Farm struct {
	Networth int
	LH       int
	XP       int
}

type Input struct {
	P           *model.Player
	Stats       *Samples
	Farm        Farm
	supportLike bool
}

func (r Input) supportPurchases() int {
	return r.P.GoldSpentWards + r.P.GoldSpentSmoke + r.P.GoldSpentDust
}

func clearlyMoreCore(a, b Input) bool {
	if a.Farm.Networth <= 0 || b.Farm.Networth <= 0 {
		return false
	}
	return a.Farm.Networth*100 > b.Farm.Networth*110
}

func coreLess(a, b Input) bool {
	if a.Farm.Networth != b.Farm.Networth {
		return a.Farm.Networth > b.Farm.Networth
	}
	if a.Farm.LH != b.Farm.LH {
		return a.Farm.LH > b.Farm.LH
	}
	if a.Farm.XP != b.Farm.XP {
		return a.Farm.XP > b.Farm.XP
	}
	if a.supportPurchases() != b.supportPurchases() {
		return a.supportPurchases() < b.supportPurchases()
	}
	return a.P.PlayerID < b.P.PlayerID
}

func AssignRoles(in []Input) {
	radiant := make([]Input, 0, 5)
	dire := make([]Input, 0, 5)
	for _, r := range in {
		if r.Stats == nil || r.Stats.total == 0 {
			continue
		}
		switch r.P.Team {
		case "radiant":
			radiant = append(radiant, r)
		case "dire":
			dire = append(dire, r)
		}
	}
	assignTeam(radiant)
	assignTeam(dire)
}

func assignTeam(in []Input) {
	byCore := func(s []Input) {
		sort.SliceStable(s, func(i, j int) bool { return coreLess(s[i], s[j]) })
	}

	ranked := append([]Input(nil), in...)
	byCore(ranked)
	coreLike := make(map[*model.Player]bool, len(in))
	for i, r := range ranked {
		coreLike[r.P] = i < 3
	}

	var mids, safes, offs, unk []Input
	for _, r := range in {
		if z, ok := r.Stats.dominant(); ok {
			switch laneOf(z, r.P.Team) {
			case laneMid:
				mids = append(mids, r)
			case laneSafe:
				safes = append(safes, r)
			case laneOff:
				offs = append(offs, r)
			default:
				unk = append(unk, r)
			}
		} else {
			unk = append(unk, r)
		}
	}
	byCore(mids)
	byCore(safes)
	byCore(offs)

	used := map[int]bool{}
	assigned := map[int]Input{}
	set := func(r Input, pos int) {
		r.P.Position = pos
		r.P.Lane = laneName(pos)
		used[pos] = true
		assigned[pos] = r
	}
	overflow := func(r Input) {
		r.supportLike = true
		unk = append(unk, r)
	}

	for i, r := range mids {
		if i == 0 && !used[laneMidPos] {
			set(r, laneMidPos)
		} else {
			overflow(r)
		}
	}
	for i, r := range safes {
		switch {
		case i == 0 && !used[safeCarryPos]:
			set(r, safeCarryPos)
		case i == 1 && !used[safeSuppPos]:
			set(r, safeSuppPos)
		default:
			overflow(r)
		}
	}
	for i, r := range offs {
		switch {
		case i == 0 && !used[offCorePos]:
			set(r, offCorePos)
		case i == 1 && !used[offSuppPos]:
			set(r, offSuppPos)
		default:
			overflow(r)
		}
	}

	byCore(unk)
	free := func(positions []int) []int {
		out := make([]int, 0, len(positions))
		for _, p := range positions {
			if !used[p] {
				out = append(out, p)
			}
		}
		return out
	}
	freeCore := free([]int{laneMidPos, safeCarryPos, offCorePos})
	freeSupp := free([]int{offSuppPos, safeSuppPos})
	take := func(slots *[]int) (int, bool) {
		if len(*slots) == 0 {
			return 0, false
		}
		p := (*slots)[0]
		*slots = (*slots)[1:]
		return p, true
	}
	for _, r := range unk {
		if !r.supportLike && coreLike[r.P] {
			if pos, ok := take(&freeCore); ok {
				set(r, pos)
				continue
			}
			if pos, ok := take(&freeSupp); ok {
				set(r, pos)
			}
			continue
		}
		if pos, ok := take(&freeSupp); ok {
			set(r, pos)
			continue
		}
		if pos, ok := take(&freeCore); ok {
			set(r, pos)
		}
	}

	if p3, ok3 := assigned[offCorePos]; ok3 {
		if p4, ok4 := assigned[offSuppPos]; ok4 && coreLike[p4.P] && clearlyMoreCore(p4, p3) {
			set(p3, offSuppPos)
			set(p4, offCorePos)
		}
	}
}

const (
	safeCarryPos = 1
	laneMidPos   = 2
	offCorePos   = 3
	offSuppPos   = 4
	safeSuppPos  = 5
)
