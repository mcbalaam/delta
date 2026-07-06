package battle

import (
	"image/color"
	"time"
)

// ── Battle states ─────────────────────────────────────────────────

type BattleState int

const (
	StateFirstCharSelecting  BattleState = iota // player picks action for party member 0
	StateFirstCharAct                           // member 0 performs ACT/spell
	StateFirstCharAttack                        // member 0 attacks (FIGHT)
	StateSecondCharSelecting                    // party member 1
	StateSecondCharAct
	StateSecondCharAttack
	StateThirdCharSelecting // party member 2
	StateThirdCharAct
	StateThirdCharAttack
	StateBoxOpen     // box entrance animation + soul flyout
	StateEnemyTarget // enemy dialogue + target selection
	StateEnemyTurn   // enemy attack — arena + soul active
	StateBoxClose    // box exit animation + soul flyback
)

func (s BattleState) IsSelecting() bool {
	return s == StateFirstCharSelecting || s == StateSecondCharSelecting || s == StateThirdCharSelecting
}

func (s BattleState) String() string {
	switch s {
	case StateFirstCharSelecting:
		return "1st Char Selecting"
	case StateFirstCharAct:
		return "1st Char Act"
	case StateFirstCharAttack:
		return "1st Char Attack"
	case StateSecondCharSelecting:
		return "2nd Char Selecting"
	case StateSecondCharAct:
		return "2nd Char Act"
	case StateSecondCharAttack:
		return "2nd Char Attack"
	case StateThirdCharSelecting:
		return "3rd Char Selecting"
	case StateThirdCharAct:
		return "3rd Char Act"
	case StateThirdCharAttack:
		return "3rd Char Attack"
	case StateBoxOpen:
		return "Box Open"
	case StateEnemyTarget:
		return "Enemy Target"
	case StateEnemyTurn:
		return "Enemy Turn"
	case StateBoxClose:
		return "Box Close"
	default:
		return "???"
	}
}

// selectingStateFor returns the selecting state for a given party index.
func selectingStateFor(idx int) BattleState {
	switch idx {
	case 0:
		return StateFirstCharSelecting
	case 1:
		return StateSecondCharSelecting
	default:
		return StateThirdCharSelecting
	}
}

func actStateFor(idx int) BattleState {
	switch idx {
	case 0:
		return StateFirstCharAct
	case 1:
		return StateSecondCharAct
	default:
		return StateThirdCharAct
	}
}

func attackStateFor(idx int) BattleState {
	switch idx {
	case 0:
		return StateFirstCharAttack
	case 1:
		return StateSecondCharAttack
	default:
		return StateThirdCharAttack
	}
}

func memberIndexFromActionState(s BattleState) int {
	switch s {
	case StateFirstCharAct, StateFirstCharAttack:
		return 0
	case StateSecondCharAct, StateSecondCharAttack:
		return 1
	default:
		return 2
	}
}

func charIndexFromSelecting(s BattleState) int {
	switch s {
	case StateFirstCharSelecting:
		return 0
	case StateSecondCharSelecting:
		return 1
	default:
		return 2
	}
}

// ── Menu states ──────────────────────────────────────────────────

type BattleMenuState int

const (
	MenuHidden BattleMenuState = iota
	MenuMain
	MenuAct
	MenuTarget
)

// ── Button IDs ───────────────────────────────────────────────────

const (
	BtnFight = iota
	BtnActMagic
	BtnItem
	BtnSpare
	BtnDefend
	BtnCount
)

// ── Supporting types ─────────────────────────────────────────────

type PillarParticle struct {
	X, Y      float64
	Dir       float64
	Accent    color.RGBA
	SpawnAt   time.Time
	CardIndex int
}

type CommittedAction struct {
	ActionType   int
	ActName      string
	TargetIdx    int
	IsAllyTarget bool
}
