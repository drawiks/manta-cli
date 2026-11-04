package replay

import (
	"math"
	"testing"
)

func TestBottleManaRestored(t *testing.T) {
	rate := bottleRegenAmount / bottleRegenDuration
	for _, tc := range []struct {
		name    string
		dt      float64
		mana    float64
		maxMana float64
		want    float64
	}{
		{name: "full tick", dt: 1, mana: 0, maxMana: 1000, want: rate},
		{name: "missing mana caps", dt: 1, mana: 95, maxMana: 100, want: 5},
		{name: "full mana", dt: 1, mana: 100, maxMana: 100, want: 0},
		{name: "one full charge", dt: bottleRegenDuration, mana: 0, maxMana: 1000, want: bottleRegenAmount},
		{name: "refreshed charges scale", dt: 3, mana: 0, maxMana: 1000, want: rate * 3},
		{name: "negative interval", dt: -1, mana: 0, maxMana: 1000, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := bottleManaRestored(tc.dt, tc.mana, tc.maxMana); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("bottleManaRestored(%v, %v, %v) = %v, want %v", tc.dt, tc.mana, tc.maxMana, got, tc.want)
			}
		})
	}
}
