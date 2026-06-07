package battle

import (
	"image/color"

	"github.com/mcbalaam/delta/internal/render"
)

// MemberState represents the party member's condition in battle.
type MemberState int

const (
	MemberAlive MemberState = iota
	MemberDown              // KO'd
)

// PartyMember is a character on the player's side.
type PartyMember struct {
	Name   string
	Sprite string // path for AnimatedIcon: "media/sprites/<name>"
	Voice  string // sound name for dialogue SFX

	// ── Battle UI ──
	AccentColor     color.Color          // colors the member's card border
	BattleMiniature *render.AnimatedIcon // character icon on the battle card

	MaxHP   float64
	HP      float64
	Attack  float64
	Defense float64
	Lv      int // optional, for damage formula scaling

	State MemberState

	IsLeader    bool
	IsDefending bool // set when this member uses DEFEND; resets each turn

	// Unique ACTs this party member can perform.
	Acts []ActDef
	// ActEffects maps an ACT name → the result when this member uses it.
	// The battle system resolves the effect (heal, shield, etc.) using this data.
	ActEffects map[string]ActEffect

	Statuses []StatusEffect

	// ── Callbacks ──────────────────────────────────────────

	// OnTurnSubmit runs when the player confirms this member's action for the turn.
	OnTurnSubmit func(battleCtx interface{})

	// OnHit runs when this member takes damage.
	OnHit func(battleCtx interface{}, damage float64)

	// OnHeal runs when this member is healed.
	OnHeal func(battleCtx interface{}, amount float64)

	// OnDown runs when HP reaches 0.
	OnDown func(battleCtx interface{})
}

// GetDialogue returns the member's name for dialogue positioning.
func (pm *PartyMember) GetDialogue() string {
	return pm.Name
}

// Alive returns true if the member is not KO'd.
func (pm *PartyMember) Alive() bool {
	return pm.State != MemberDown && pm.HP > 0
}

// TakeDamage reduces HP by damage (after defense), triggers OnHit, and checks for down.
func (pm *PartyMember) TakeDamage(battleCtx interface{}, rawDamage float64) float64 {
	_, def, _ := pm.EffectiveStats()
	damage := rawDamage / def
	if damage < 1 {
		damage = 1
	}
	pm.HP -= damage
	if pm.HP <= 0 {
		pm.HP = 0
		pm.State = MemberDown
		if pm.OnDown != nil {
			pm.OnDown(battleCtx)
		}
	}
	if pm.OnHit != nil {
		pm.OnHit(battleCtx, damage)
	}
	return damage
}

// Heal restores HP and triggers OnHeal.
func (pm *PartyMember) Heal(battleCtx interface{}, amount float64) {
	pm.HP += amount
	if pm.HP > pm.MaxHP {
		pm.HP = pm.MaxHP
	}
	if pm.State == MemberDown && pm.HP > 0 {
		pm.State = MemberAlive
	}
	if pm.OnHeal != nil {
		pm.OnHeal(battleCtx, amount)
	}
}

// ApplyStatus adds a status effect or refreshes it.
func (pm *PartyMember) ApplyStatus(se *StatusEffect) {
	for i := range pm.Statuses {
		if pm.Statuses[i].Name == se.Name {
			pm.Statuses[i] = *se
			return
		}
	}
	pm.Statuses = append(pm.Statuses, *se)
}

// TickStatuses decrements durations; removes expired ones.
func (pm *PartyMember) TickStatuses() {
	alive := pm.Statuses[:0]
	for _, s := range pm.Statuses {
		if s.Duration < 0 {
			alive = append(alive, s)
			continue
		}
		s.Duration--
		if s.Duration > 0 {
			alive = append(alive, s)
		}
	}
	pm.Statuses = alive
}

// EffectiveStats returns Attack/Defense/MercyMult after status modifiers.
func (pm *PartyMember) EffectiveStats() (attack, defense, mercyMult float64) {
	attack = pm.Attack
	defense = pm.Defense
	mercyMult = 1.0
	for _, s := range pm.Statuses {
		attack *= s.Modifier.AttackMod
		defense *= s.Modifier.DefenseMod
		mercyMult *= s.Modifier.MercyMod
	}
	return
}

// ActEffect defines what happens when this party member uses a specific ACT.
type ActEffect struct {
	HealAmount    float64       // HP restored
	StatusApply   *StatusEffect // status to apply to target (or self)
	TargetAlly    bool          // true = affects self/ally, false = affects opponent
	DialogueLines []string      // flavour text shown when used
}
