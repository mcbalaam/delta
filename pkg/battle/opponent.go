package battle

// OpponentState represents the visual/emotional state of an opponent.
type OpponentState int

const (
	StateNeutral   OpponentState = iota
	StateTired                   // near spare threshold
	StateFlustered               // vulnerable to spare
	StateDefeated                // defeated / spared
)

// StatusMod describes stat modifications from a status effect.
type StatusMod struct {
	AttackMod  float64 // multiplier (1.0 = normal)
	DefenseMod float64 // multiplier (1.0 = normal)
	MercyMod   float64 // multiplier for mercy gain (1.0 = normal)
}

// StatusEffect is a temporary effect applied to an opponent.
type StatusEffect struct {
	Name     string
	Duration int // turns remaining; -1 = permanent
	Modifier StatusMod
}

// ActDef describes a unique ACT action available for this opponent.
type ActDef struct {
	Name        string
	Description string // shown in the ACT menu
	TargetSelf  bool   // false = targets this opponent, true = targets party
}

// ActReaction defines how the opponent responds to an ACT.
type ActReaction struct {
	Dialogue    []string      // lines spoken in a speech bubble
	StateChange OpponentState // new state after the act (or -1 to keep)
	MercyAmount float64       // mercy gained
	StatusApply *StatusEffect // status to apply (nil = none)
	AttackDelta float64       // change to Attack stat (can be negative)
}

// Opponent is a single enemy in a battle encounter.
type Opponent struct {
	Name   string
	Sprite string // path for AnimatedIcon: "media/sprites/<name>"
	Voice  string // sound name for dialogue SFX

	MaxHP   float64
	HP      float64
	Attack  float64
	Defense float64

	MaxMercy float64
	Mercy    float64

	State OpponentState

	// Unique ACTs the player can use against this opponent.
	Acts []ActDef
	// Reactions maps an ACT name → the opponent's response.
	Reactions map[string]ActReaction

	Statuses []StatusEffect

	// ── Callbacks ──────────────────────────────────────────

	// OnTurnStart runs at the beginning of this opponent's turn.
	OnTurnStart func(battleCtx interface{})

	// OnDefeat runs when the opponent is defeated or spared.
	OnDefeat func(battleCtx interface{})

	// OnHit runs every time the opponent takes damage.
	OnHit func(battleCtx interface{}, damage float64)

	// AttackPattern returns a custom attack sequence for this opponent's turn.
	// If nil, the default pattern from the Turn script is used.
	AttackPattern func() *AttackSequence
}

// GetDialogue returns the opponent's name for dialogue positioning.
func (o *Opponent) GetDialogue() string {
	return o.Name
}

// Alive returns true if the opponent is not defeated.
func (o *Opponent) Alive() bool {
	return o.State != StateDefeated && o.HP > 0
}

// ApplyStatus adds a status effect or refreshes it if already active.
func (o *Opponent) ApplyStatus(se *StatusEffect) {
	for i := range o.Statuses {
		if o.Statuses[i].Name == se.Name {
			o.Statuses[i] = *se // refresh
			return
		}
	}
	o.Statuses = append(o.Statuses, *se)
}

// TickStatuses decrements durations; removes expired ones.
func (o *Opponent) TickStatuses() {
	alive := o.Statuses[:0]
	for _, s := range o.Statuses {
		if s.Duration < 0 {
			alive = append(alive, s)
			continue
		}
		s.Duration--
		if s.Duration > 0 {
			alive = append(alive, s)
		}
	}
	o.Statuses = alive
}

// EffectiveStats returns Attack/Defense/Mercy after status modifiers.
func (o *Opponent) EffectiveStats() (attack, defense, mercyMult float64) {
	attack = o.Attack
	defense = o.Defense
	mercyMult = 1.0
	for _, s := range o.Statuses {
		attack *= s.Modifier.AttackMod
		defense *= s.Modifier.DefenseMod
		mercyMult *= s.Modifier.MercyMod
	}
	return
}
