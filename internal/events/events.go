package events

type PlayerInfo struct {
	Slot           int
	SteamID        int64
	Hero           string
	Name           string
	Team           string
	TowerDamage    int
	HasTowerDamage bool
	GoldLost       int
	HasGoldLost    bool
}

type Epilogue struct {
	MatchID     int64
	DurationSec float64
	RadiantWin  bool
	Players     []PlayerInfo
}

type Slot struct {
	Slot int
	Team string
	Hero string
}

type Interval struct {
	Slot             int
	Time             int
	Unit             string
	LifeState        int
	HasLifeState     bool
	Gold             int
	TowerDamage      int
	LastHits         int
	XP               int
	X, Y             float64
	HasPos           bool
	Stuns            float64
	HeroID           int
	Level            int
	Kills            int
	Deaths           int
	Assists          int
	CampsStacked     int
	CreepsStacked    int
	RunePickups      int
	WisdomsCaptured  int
	WatchersCaptured int
	LotusesGathered  int
	SentryWards      int
	Networth         int
}

type CombatType uint8

const (
	CombatDamage CombatType = iota
	CombatHeal
	CombatModifierAdd
	CombatModifierRemove
	CombatPurchase
	CombatGold
	CombatDeath
	CombatAbility
	CombatManaRestore
)

type Combat struct {
	Time             int
	Type             CombatType
	AttackerName     string
	TargetName       string
	SourceName       string
	TargetSourceName string
	Inflictor        string
	ValueName        string
	AttackerHero     bool
	TargetHero       bool
	AttackerIllusion bool
	TargetIllusion   bool
	Value            int
	StunDuration     float64
	GoldReason       int
	AbilityLevel     int
}

type ChatKind uint8

const (
	ChatFirstBlood ChatKind = iota
	ChatMiniboss
)

type Chat struct {
	Kind ChatKind
	Slot int
	Time int
}

type AbilityLevel struct {
	Hero    string
	Ability string
	Level   int
	Time    int
}

type Handler interface {
	Epilogue(Epilogue)
	Slot(Slot)
	Interval(Interval)
	Combat(Combat)
	Chat(Chat)
	AbilityLevel(AbilityLevel)
}
