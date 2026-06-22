package battle

import (
	"time"

	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/systems"
)

// DialogueEmitter identifies who is speaking (enemy or party member).
type DialogueEmitter interface {
	GetDialogue() string
}

// TurnEvent is a single step in the enemy's turn sequence.
// Run executes the step. Must call onDone when finished.
type TurnEvent interface {
	Run(battle *Battle, onDone func())
}

// NarrativeEvent shows text in the narrative style (top of screen).
type NarrativeEvent struct {
	Lines []string
}

func (e *NarrativeEvent) Run(b *Battle, onDone func()) {
	session := engine.NewDialogueSession(
		b.TextEngine,
		e.Lines,
		engine.StyleNarrative,
		b.SoundPlayer,
	)
	session.OnAllComplete = onDone
	b.turnSession = session
	session.Start()
}

// DialogueEvent shows speech from a specific character.
type DialogueEvent struct {
	Emitter DialogueEmitter
	Lines   []string
}

func (e *DialogueEvent) Run(b *Battle, onDone func()) {
	session := engine.NewDialogueSession(
		b.TextEngine,
		e.Lines,
		engine.StyleBubble,
		b.SoundPlayer,
	)
	session.OnAllComplete = func() {
		session.Destroy()
		onDone()
	}
	b.turnSession = session
	session.Start()
}

// WaitEvent pauses the sequence.
type WaitEvent struct {
	Duration time.Duration // 0 = wait for Z press
}

func (e *WaitEvent) Run(b *Battle, onDone func()) {
	if e.Duration <= 0 {
		b.turnWaitingForZ = true
		b.turnWaitCallback = onDone
		return
	}
	b.turnWaitingForTimer = true
	b.turnTimerTarget = e.Duration
	b.turnTimerElapsed = 0
	b.turnWaitCallback = onDone
}

// AnimationEvent plays an animation on a character. Passes through the signal bus.
type AnimationEvent struct {
	Emitter  DialogueEmitter
	AnimName string
}

func (e *AnimationEvent) Run(b *Battle, onDone func()) {
	systems.MasterSignalBus.Emit("battle_animation", nil,
		map[string]interface{}{
			"emitter": e.Emitter,
			"anim":    e.AnimName,
			"on_done": onDone,
		})
}

// AttackEvent launches the attack sequence and waits for it to finish.
type AttackEvent struct {
	Sequence *AttackSequence
	Duration time.Duration
}

func (e *AttackEvent) Run(b *Battle, onDone func()) {
	b.turnAttackSeq = e.Sequence
	b.turnAttackDuration = e.Duration
	b.turnAttackElapsed = 0
	b.turnAttackDone = onDone

	b.SetState(StateEnemyTurn)
}

// Turn is the script for everything that happens after the player confirms actions.
type Turn struct {
	Sequence []TurnEvent
}

// ── TurnPlayer ───────────────────────────────────────────────────

type TurnPlayer struct {
	turn   *Turn
	battle *Battle
	index  int
}

func NewTurnPlayer(b *Battle, turn *Turn) *TurnPlayer {
	return &TurnPlayer{
		turn:   turn,
		battle: b,
		index:  0,
	}
}

func (tp *TurnPlayer) Start() {
	tp.index = 0
	tp.next()
}

func (tp *TurnPlayer) next() {
	if tp.battle.turnSession != nil {
		tp.battle.turnSession.Destroy()
		tp.battle.turnSession = nil
	}

	if tp.index >= len(tp.turn.Sequence) {
		tp.battle.Targets = nil
		if tp.battle.OnTurnComplete != nil {
			tp.battle.OnTurnComplete()
		}
		return
	}

	event := tp.turn.Sequence[tp.index]
	tp.index++
	event.Run(tp.battle, func() { tp.next() })
}
