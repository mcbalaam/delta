package battle

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
)

// ── Battle ───────────────────────────────────────────────────────

type Battle struct {
	State       BattleState
	TextEngine  *engine.TextEngine
	SoundPlayer *sound.SoundPlayer

	Party        []*PartyMember
	Opponents    []*Opponent
	ActiveMember int

	MenuState      BattleMenuState
	SelectedButton int
	SelectedAct    int
	SelectedTarget int
	targetIsAlly   bool
	PendingActName string

	MenuSprite *render.AnimatedIcon
	SoulSprite *render.AnimatedIcon
	HPIcon     *render.AnimatedIcon
	SlashIcon  *render.AnimatedIcon

	turnPlayer          *TurnPlayer
	turnSession         *engine.DialogueSession
	turnWaitingForZ     bool
	turnWaitCallback    func()
	turnWaitingForTimer bool
	turnTimerTarget     time.Duration
	turnTimerElapsed    time.Duration
	turnAttackSeq       *AttackSequence
	turnAttackDuration  time.Duration
	turnAttackElapsed   time.Duration
	turnAttackDone      func()

	Targets    []int
	TargetIcon *render.AnimatedIcon

	DialogueBoxIcon *render.AnimatedIcon

	stateDebugNotice *engine.DebugNotice

	narrativeLines []string
	restoredText   *engine.TextDisplay

	CommittedActions []*CommittedAction

	cardAnimY []float64

	pillarParticles   []PillarParticle
	lastParticleSpawn time.Time

	flickerAccum float64 // for opponent sprite flicker during target selection

	showArena      func()
	hideArena      func()
	startExitArena func()
	ArenaBoundsX   float64
	ArenaBoundsY   float64
	ArenaBoundsW   float64
	ArenaBoundsH   float64

	SoulX        float64
	SoulY        float64
	SoulCollider *components.Collider

	soulFlyoutPlaying  bool
	soulFlyoutProgress float64
	soulFlyoutDuration float64
	soulFlyoutStartX   float64
	soulFlyoutStartY   float64
	soulFlyoutTargetX  float64
	soulFlyoutTargetY  float64

	boxOpenTimer     float64
	boxOpenDuration  float64
	boxCloseTimer    float64
	boxCloseDuration float64

	soulFlybackStartX  float64
	soulFlybackStartY  float64
	soulFlybackTargetX float64
	soulFlybackTargetY float64

	InvincibilityTimer    float64
	InvincibilityDuration float64

	OnTurnComplete func()

	ShowHitboxes bool
}

// ── Setters ──────────────────────────────────────────────────────

func (b *Battle) SetMenuSprite(s *render.AnimatedIcon) { b.MenuSprite = s }
func (b *Battle) SetSoulSprite(s *render.AnimatedIcon) {
	b.SoulSprite = s
	b.SoulCollider = components.NewCollider(20, 20, 0, 0)
	b.InvincibilityDuration = 1
}
func (b *Battle) SetArenaHooks(show, hide func()) { b.showArena = show; b.hideArena = hide }
func (b *Battle) SetStartExitArena(fn func())     { b.startExitArena = fn }
func (b *Battle) SetArenaBounds(x, y, w, h float64) {
	b.ArenaBoundsX, b.ArenaBoundsY, b.ArenaBoundsW, b.ArenaBoundsH = x, y, w, h
}
func (b *Battle) SetTargetIcon(s *render.AnimatedIcon)  { b.TargetIcon = s }
func (b *Battle) SetDialogueBox(s *render.AnimatedIcon) { b.DialogueBoxIcon = s }

// ── Helpers ──────────────────────────────────────────────────────

func (b *Battle) ButtonLabel(btn int) string {
	switch btn {
	case BtnFight:
		return "FIGHT"
	case BtnActMagic:
		return b.ActMagicLabel()
	case BtnItem:
		return "ITEM"
	case BtnSpare:
		return "SPARE"
	case BtnDefend:
		return "DEFEND"
	}
	return ""
}

func (b *Battle) ActMagicLabel() string {
	if b.ActiveMember < 0 || b.ActiveMember >= len(b.Party) {
		return "ACT"
	}
	m := b.Party[b.ActiveMember]
	if m != nil && m.IsLeader {
		return "ACT"
	}
	return "MAGIC"
}

func (b *Battle) CollectActs() []ActEntry {
	var active *PartyMember
	if len(b.Party) > 0 {
		active = b.Party[b.ActiveMember]
	}
	var target *Opponent
	if len(b.Opponents) > 0 {
		target = b.Opponents[0]
	}
	return CollectActs(active, target)
}

// ── State machine core ───────────────────────────────────────────

// SetState transitions to a new battle state, handling all enter/exit side effects.
func (b *Battle) SetState(s BattleState) {
	switch b.State {
	case StateEnemyTurn:
		b.turnAttackSeq = nil
		b.Targets = nil
	}

	b.completeStateTransition(s)
}

func (b *Battle) completeStateTransition(s BattleState) {
	b.State = s
	b.MenuState = MenuHidden

	if b.stateDebugNotice != nil {
		b.stateDebugNotice.Destroy()
	}
	b.stateDebugNotice = engine.ShowDebugNotice(b.TextEngine, "BATTLE STATE: "+s.String(), 10, -10, 999*time.Hour)

	switch s {
	case StateFirstCharSelecting, StateSecondCharSelecting, StateThirdCharSelecting:
		b.enterCharSelecting(s)
	case StateFirstCharAct, StateSecondCharAct, StateThirdCharAct:
		b.enterCharAct(memberIndexFromActionState(s))
	case StateFirstCharAttack, StateSecondCharAttack, StateThirdCharAttack:
		b.enterCharAttack(memberIndexFromActionState(s))
	case StateBoxOpen:
		b.enterBoxOpen()
	case StateEnemyTarget:
		b.enterEnemyTarget()
	case StateEnemyTurn:
		b.enterEnemyTurn()
	case StateBoxClose:
		b.enterBoxClose()
	}
}

func (b *Battle) enterCharSelecting(s BattleState) {
	idx := charIndexFromSelecting(s)
	b.ActiveMember = idx
	b.MenuState = MenuMain
	b.SelectedButton = 0
	if b.turnSession == nil {
		b.restoreNarrative()
	}
}

func (b *Battle) enterCharAct(memberIdx int) {
	action := b.CommittedActions[memberIdx]
	if action == nil {
		b.advanceFromAction()
		return
	}
	member := b.Party[memberIdx]
	b.executeAct(member, action, func() { b.advanceFromAction() })
}

func (b *Battle) enterCharAttack(memberIdx int) {
	b.advanceFromAction()
}

// advanceFromAction advances to the next executed member's action or to EnemyTarget.
func (b *Battle) advanceFromAction() {
	memberIdx := memberIndexFromActionState(b.State)

	for i := memberIdx + 1; i < len(b.Party); i++ {
		if !b.Party[i].Alive() {
			continue
		}
		action := b.CommittedActions[i]
		if action == nil {
			continue
		}
		switch action.ActionType {
		case BtnActMagic:
			b.SetState(actStateFor(i))
			return
		case BtnFight:
			b.SetState(attackStateFor(i))
			return
		}
	}

	b.SetState(StateEnemyTarget)
}

// advanceFromSelecting moves to the next alive member's selecting state.
// All members select first, then executions begin.
func (b *Battle) advanceFromSelecting() {
	next := b.ActiveMember + 1
	for next < len(b.Party) {
		if b.Party[next].Alive() {
			b.SetState(selectingStateFor(next))
			return
		}
		next++
	}

	b.startExecutions()
}

func (b *Battle) startExecutions() {
	for i := 0; i < len(b.Party); i++ {
		if !b.Party[i].Alive() {
			continue
		}
		action := b.CommittedActions[i]
		if action == nil {
			continue
		}
		switch action.ActionType {
		case BtnActMagic:
			b.SetState(actStateFor(i))
			return
		case BtnFight:
			b.SetState(attackStateFor(i))
			return
		}
	}

	b.SetState(StateEnemyTarget)
}

func (b *Battle) undoLastMember() {
	current := b.State
	var prevIdx int
	switch current {
	case StateSecondCharSelecting:
		prevIdx = 0
	case StateThirdCharSelecting:
		prevIdx = 1
	default:
		return
	}

	m := b.Party[prevIdx]
	if m != nil {
		m.IsDefending = false
	}
	b.CommittedActions[prevIdx] = nil
	if b.SoundPlayer != nil {
		b.SoundPlayer.PlaySound("smallswing", 1.0)
	}
	b.SetState(selectingStateFor(prevIdx))
}

func (b *Battle) enterEnemyTarget() {
	b.selectTargets()
	if b.turnPlayer != nil && len(b.turnPlayer.turn.Sequence) > 0 {
		b.turnPlayer.Start()
	}
}

func (b *Battle) enterEnemyTurn() {
	// arena already shown by BoxOpen, soul already at center
}

// ── Start turn ───────────────────────────────────────────────────

func (b *Battle) StartTurn(turn *Turn, narrativeLines []string) {
	for _, m := range b.Party {
		m.IsDefending = false
		if m.CharacterSprite != nil {
			m.CharacterSprite.SetIconState("idle")
			m.CharacterSprite.CurrentState.Mode = render.AnimationModeLoop
		}
	}

	b.SelectedAct = 0
	b.SelectedTarget = 0
	b.targetIsAlly = false
	b.PendingActName = ""
	b.CommittedActions = make([]*CommittedAction, len(b.Party))
	b.turnPlayer = NewTurnPlayer(b, turn)
	b.narrativeLines = narrativeLines

	if len(narrativeLines) > 0 {
		session := engine.NewDialogueSession(
			b.TextEngine, narrativeLines, engine.StyleNarrative, b.SoundPlayer,
		)
		b.turnSession = session
		session.Start()
	}

	if b.State == StateEnemyTurn || b.State == StateEnemyTarget {
		b.SetState(StateBoxClose)
		return
	}

	for i := 0; i < len(b.Party); i++ {
		if b.Party[i].Alive() {
			b.SetState(selectingStateFor(i))
			return
		}
	}
	b.SetState(StateEnemyTarget)
}

// ── Execute helpers ──────────────────────────────────────────────

func (b *Battle) executeAct(member *PartyMember, action *CommittedAction, onDone func()) {
	if member.CharacterSprite != nil {
		if err := member.CharacterSprite.SetIconState("act"); err != nil {
			member.CharacterSprite.SetIconState("idle")
		}
		member.CharacterSprite.CurrentState.Mode = render.AnimationModeOnce
	}

	narrative := b.resolveActEffect(member, action)

	b.showActionNarrative(narrative, func() {
		if member.CharacterSprite != nil {
			member.CharacterSprite.SetIconState("idle")
			member.CharacterSprite.CurrentState.Mode = render.AnimationModeLoop
		}
		onDone()
	})
}

func (b *Battle) executeItem(member *PartyMember, action *CommittedAction, onDone func()) {
	b.showActionNarrative("* "+member.Name+" used an item!$f", onDone)
}

func (b *Battle) resolveActEffect(member *PartyMember, action *CommittedAction) string {
	var actInstance Act

	if action.IsAllyTarget && action.TargetIdx >= 0 && action.TargetIdx < len(b.Party) {
		ally := b.Party[action.TargetIdx]
		for _, a := range ally.Acts {
			if a.Name() == action.ActName {
				actInstance = a
				break
			}
		}
	}

	if actInstance == nil {
		for _, a := range b.Party[b.ActiveMember].Acts {
			if a.Name() == action.ActName {
				actInstance = a
				break
			}
		}
	}

	if actInstance != nil {
		ctx := &ActContext{
			Battle:       b,
			ActiveMember: member,
			TargetIdx:    action.TargetIdx,
			IsAllyTarget: action.IsAllyTarget,
		}
		if action.IsAllyTarget && action.TargetIdx >= 0 && action.TargetIdx < len(b.Party) {
			ctx.Target = b.Party[action.TargetIdx]
		} else if action.TargetIdx >= 0 && action.TargetIdx < len(b.Opponents) {
			ctx.Target = b.Opponents[action.TargetIdx]
		} else {
			ctx.Target = b.Opponents[0]
		}

		result := actInstance.Execute(ctx)

		// Also apply reactions if there are any
		if !action.IsAllyTarget && action.TargetIdx >= 0 && action.TargetIdx < len(b.Opponents) {
			opp := b.Opponents[action.TargetIdx]
			if opp.Reactions != nil {
				if react, ok := opp.Reactions[action.ActName]; ok {
					if react.MercyAmount > 0 {
						opp.Mercy += react.MercyAmount
					}
					if react.StateChange != -1 {
						opp.State = react.StateChange
					}
					if react.AttackDelta != 0 {
						opp.Attack += react.AttackDelta
					}
					if react.StatusApply != nil {
						opp.ApplyStatus(react.StatusApply)
					}
				}
			}
		}

		return result
	}

	return "* " + member.Name + " used " + action.ActName + "!$e"
}

func (b *Battle) commitAction(actionType int, actName string, targetIdx int, isAllyTarget bool) {
	if b.ActiveMember < 0 || b.ActiveMember >= len(b.Party) {
		return
	}
	b.CommittedActions[b.ActiveMember] = &CommittedAction{
		ActionType:   actionType,
		ActName:      actName,
		TargetIdx:    targetIdx,
		IsAllyTarget: isAllyTarget,
	}
}

// ── Update ───────────────────────────────────────────────────────

func (b *Battle) Update(dt time.Duration) {
	b.updateCharacterAnimations(dt)

	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		b.ShowHitboxes = !b.ShowHitboxes
	}

	if b.State.IsSelecting() && b.MenuState == MenuTarget && !b.targetIsAlly {
		b.flickerAccum += dt.Seconds()
	}

	if b.TargetIcon != nil {
		b.TargetIcon.Update(dt)
	}

	switch {
	case b.State.IsSelecting():
		return

	case b.State == StateFirstCharAct || b.State == StateSecondCharAct || b.State == StateThirdCharAct:
		return

	case b.State == StateFirstCharAttack || b.State == StateSecondCharAttack || b.State == StateThirdCharAttack:
		return

	case b.State == StateBoxOpen:
		b.updateSoulMovement(dt)
		b.updateBoxOpen(dt)
		return

	case b.State == StateEnemyTarget:
		b.updateTurnWait(dt)
		return

	case b.State == StateEnemyTurn:
		b.updateSoulMovement(dt)
		b.updateTurnWait(dt)
		b.updateAttack(dt)
		return

	case b.State == StateBoxClose:
		b.updateBoxClose(dt)
		return
	}
}
