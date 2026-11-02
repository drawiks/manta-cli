package stats

import "sort"

// Lane and position detection uses the first roleWindowSec seconds of hero
// world positions plus the farm profile at the end of that window.
const (
	roleWindowSec = 600.0

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

type roleSamples struct {
	zones [4]int
	total int
}

func (r *roleSamples) sample(x, y float64) {
	r.zones[classifyZone(x, y)]++
	r.total++
}

func (r *roleSamples) dominant() (roleZone, bool) {
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

type roleFarm struct {
	networth int
	lh       int
	xp       int
}

type roleInput struct {
	p           *Player
	stats       *roleSamples
	farm        roleFarm
	supportLike bool
}

func (r roleInput) supportPurchases() int {
	return r.p.GoldSpentWards + r.p.GoldSpentSmoke + r.p.GoldSpentDust
}

func clearlyMoreCore(a, b roleInput) bool {
	if a.farm.networth <= 0 || b.farm.networth <= 0 {
		return false
	}
	return a.farm.networth*100 > b.farm.networth*110
}

func coreLess(a, b roleInput) bool {
	if a.farm.networth != b.farm.networth {
		return a.farm.networth > b.farm.networth
	}
	if a.farm.lh != b.farm.lh {
		return a.farm.lh > b.farm.lh
	}
	if a.farm.xp != b.farm.xp {
		return a.farm.xp > b.farm.xp
	}
	if a.supportPurchases() != b.supportPurchases() {
		return a.supportPurchases() < b.supportPurchases()
	}
	return a.p.PlayerID < b.p.PlayerID
}

func assignRoles(in []roleInput) {
	radiant := make([]roleInput, 0, 5)
	dire := make([]roleInput, 0, 5)
	for _, r := range in {
		if r.stats == nil || r.stats.total == 0 {
			continue
		}
		switch r.p.Team {
		case "radiant":
			radiant = append(radiant, r)
		case "dire":
			dire = append(dire, r)
		}
	}
	assignTeam(radiant)
	assignTeam(dire)
}

func assignTeam(in []roleInput) {
	byCore := func(s []roleInput) {
		sort.SliceStable(s, func(i, j int) bool { return coreLess(s[i], s[j]) })
	}

	ranked := append([]roleInput(nil), in...)
	byCore(ranked)
	coreLike := make(map[*Player]bool, len(in))
	for i, r := range ranked {
		coreLike[r.p] = i < 3
	}

	var mids, safes, offs, unk []roleInput
	for _, r := range in {
		if z, ok := r.stats.dominant(); ok {
			switch laneOf(z, r.p.Team) {
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
	assigned := map[int]roleInput{}
	set := func(r roleInput, pos int) {
		r.p.Position = pos
		r.p.Lane = laneName(pos)
		used[pos] = true
		assigned[pos] = r
	}
	overflow := func(r roleInput) {
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
		if !r.supportLike && coreLike[r.p] {
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
		if p4, ok4 := assigned[offSuppPos]; ok4 && coreLike[p4.p] && clearlyMoreCore(p4, p3) {
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
