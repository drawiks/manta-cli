package replay

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/dotabuff/manta/dota"

	"github.com/drawiks/manta-cli/internal/events"
)

func TestParseStreamClassifiesBadMagic(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte("not a replay"),
		[]byte("PUFDEMS\x00"),
	} {
		err := ParseStream(context.Background(), bytes.NewReader(data), nopHandler{})
		if !errors.Is(err, ErrNotReplay) {
			t.Errorf("ParseStream(%q) = %v, want ErrNotReplay", data, err)
		}
	}
}

func TestParseStreamClassifiesMissingPlayerResource(t *testing.T) {
	header := make([]byte, 16)
	copy(header, replayMagic)

	err := ParseStream(context.Background(), bytes.NewReader(header), nopHandler{})
	if !errors.Is(err, ErrNoPlayerResource) {
		t.Fatalf("ParseStream(header only) = %v, want ErrNoPlayerResource", err)
	}
}

func TestCombatTypeManaRestore(t *testing.T) {
	got, ok := combatType(dota.DOTA_COMBATLOG_TYPES_DOTA_COMBATLOG_MANA_RESTORED)
	if !ok || got != events.CombatManaRestore {
		t.Fatalf("combatType(mana restored) = %v, %v; want %v, true", got, ok, events.CombatManaRestore)
	}
}
