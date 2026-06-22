package battle

import (
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
)

// ── Battle states ─────────────────────────────────────────────────

type BattleState int

const (
	StateFirstCharSelecting  BattleState = iota // player picks action for party member 0
	StateFirstCharExecute                       // executing party member 0's action
	StateSecondCharSelecting                    // party member 1
	StateSecondCharExecute
	StateThirdCharSelecting // party member 2
	StateThirdCharExecute
	StateEnemyTarget // enemy dialogue + target selection
	StateEnemyTurn   // enemy attack — arena + soul active
)

func (s BattleState) IsSelecting() bool {
	return s == StateFirstCharSelecting || s == StateSecondCharSelecting || s == StateThirdCharSelecting
}

func (s BattleState) IsExecuting() bool {
	return s == StateFirstCharExecute || s == StateSecondCharExecute || s == StateThirdCharExecute
}

func (s BattleState) String() string {
	switch s {
	case StateFirstCharSelecting:
		return "1st Char Selecting"
	case StateFirstCharExecute:
		return "1st Char Execute"
	case StateSecondCharSelecting:
		return "2nd Char Selecting"
	case StateSecondCharExecute:
		return "2nd Char Execute"
	case StateThirdCharSelecting:
		return "3rd Char Selecting"
	case StateThirdCharExecute:
		return "3rd Char Execute"
	case StateEnemyTarget:
		return "Enemy Target"
	case StateEnemyTurn:
		return "Enemy Turn"
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

// executionStateFor returns the execution state for a given party index.
func executionStateFor(idx int) BattleState {
	switch idx {
	case 0:
		return StateFirstCharExecute
	case 1:
		return StateSecondCharExecute
	default:
		return StateThirdCharExecute
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

// PillarParticle is a vertical bar spawned from card pillar tips.
type PillarParticle struct {
	X, Y      float64
	Dir       float64
	Accent    color.RGBA
	SpawnAt   time.Time
	CardIndex int
}

// CommittedAction records what a party member chose to do this turn.
type CommittedAction struct {
	ActionType   int
	ActName      string
	TargetIdx    int
	IsAllyTarget bool
}

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

	showArena    func()
	hideArena    func()
	ArenaBoundsX float64
	ArenaBoundsY float64
	ArenaBoundsW float64
	ArenaBoundsH float64

	SoulX        float64
	SoulY        float64
	SoulCollider *components.Collider

	OnTurnComplete func()

	ShowHitboxes bool
}

// ── Setters ──────────────────────────────────────────────────────

func (b *Battle) SetMenuSprite(s *render.AnimatedIcon) { b.MenuSprite = s }
func (b *Battle) SetSoulSprite(s *render.AnimatedIcon) {
	b.SoulSprite = s
	b.SoulCollider = components.NewCollider(20, 20, 0, 0)
}
func (b *Battle) SetArenaHooks(show, hide func()) { b.showArena = show; b.hideArena = hide }
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
		if b.hideArena != nil {
			b.hideArena()
		}
	case StateFirstCharExecute, StateSecondCharExecute, StateThirdCharExecute:
		b.clearNarrativeText()
	}

	b.State = s
	b.MenuState = MenuHidden

	if b.stateDebugNotice != nil {
		b.stateDebugNotice.Destroy()
	}
	b.stateDebugNotice = engine.ShowDebugNotice(b.TextEngine, "BATTLE STATE: "+s.String(), 10, -10, 999*time.Hour)

	switch s {
	case StateFirstCharSelecting, StateSecondCharSelecting, StateThirdCharSelecting:
		b.enterCharSelecting(s)
	case StateFirstCharExecute, StateSecondCharExecute, StateThirdCharExecute:
		b.enterCharExecute(s)
	case StateEnemyTarget:
		b.enterEnemyTarget()
	case StateEnemyTurn:
		b.enterEnemyTurn()
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

func (b *Battle) enterCharExecute(s BattleState) {
	idx := charIndexFromExecute(s)
	action := b.CommittedActions[idx]
	if action == nil {
		b.advanceFromExecute()
		return
	}

	member := b.Party[idx]
	switch action.ActionType {
	case BtnActMagic:
		b.executeAct(member, action, func() { b.advanceFromExecute() })
	case BtnItem:
		b.executeItem(member, action, func() { b.advanceFromExecute() })
	default:
		b.advanceFromExecute()
	}
}

func (b *Battle) advanceFromExecute() {
	current := b.State
	var nextIdx int
	switch current {
	case StateFirstCharExecute:
		nextIdx = 1
	case StateSecondCharExecute:
		nextIdx = 2
	default:
		b.SetState(StateEnemyTarget)
		return
	}

	// Move to the next party member's execute state
	for i := nextIdx; i < len(b.Party); i++ {
		if b.Party[i].Alive() {
			b.SetState(executionStateFor(i))
			return
		}
	}
	b.SetState(StateEnemyTarget)
}

// advanceFromSelecting moves to the next selecting state or starts execution.
func (b *Battle) advanceFromSelecting() {
	next := b.ActiveMember + 1
	for next < len(b.Party) {
		if b.Party[next].Alive() {
			b.SetState(selectingStateFor(next))
			return
		}
		next++
	}
	// All members have selected — start execution phase for first alive member
	for i := 0; i < len(b.Party); i++ {
		if b.Party[i].Alive() {
			b.SetState(executionStateFor(i))
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
	if m != nil && m.IsDefending {
		m.IsDefending = false
		if m.CharacterSprite != nil {
			m.PlayingDefendRev = true
			m.DefendRevFrame = m.CharacterSprite.CurrentState.CurrentFrame
		}
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
	if b.showArena != nil {
		b.showArena()
	}
	if b.ArenaBoundsW > 0 && b.ArenaBoundsH > 0 {
		b.SoulX = b.ArenaBoundsX + b.ArenaBoundsW/2
		b.SoulY = b.ArenaBoundsY + b.ArenaBoundsH/2
	}
}

// ── State index helpers ──────────────────────────────────────────

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

func charIndexFromExecute(s BattleState) int {
	switch s {
	case StateFirstCharExecute:
		return 0
	case StateSecondCharExecute:
		return 1
	default:
		return 2
	}
}

// ── Start turn ───────────────────────────────────────────────────

func (b *Battle) StartTurn(turn *Turn, narrativeLines []string) {
	for _, m := range b.Party {
		m.IsDefending = false
		m.PlayingDefendRev = false
		m.DefendRevFrame = 0
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

	// Enter the first selecting state for the first alive member
	for i := 0; i < len(b.Party); i++ {
		if b.Party[i].Alive() {
			b.SetState(selectingStateFor(i))
			return
		}
	}
	// No alive members
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
	if member.ActEffects != nil {
		if ef, ok := member.ActEffects[action.ActName]; ok {
			if ef.HealAmount > 0 || ef.StatusApply != nil {
				target := member
				if action.IsAllyTarget && action.TargetIdx >= 0 && action.TargetIdx < len(b.Party) {
					target = b.Party[action.TargetIdx]
				}
				if ef.HealAmount > 0 {
					target.Heal(b, ef.HealAmount)
				}
				if ef.StatusApply != nil {
					target.ApplyStatus(ef.StatusApply)
				}
			}
			if len(ef.DialogueLines) > 0 {
				return ef.DialogueLines[0]
			}
		}
	}

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
				if len(react.Dialogue) > 0 {
					return react.Dialogue[0]
				}
			}
		}
	}

	return "* " + member.Name + " used " + action.ActName + "!$f"
}

func (b *Battle) showActionNarrative(text string, onDone func()) {
	if b.turnSession != nil {
		b.turnSession.Destroy()
	}
	session := engine.NewDialogueSession(
		b.TextEngine,
		[]string{text},
		engine.StyleNarrative,
		b.SoundPlayer,
	)
	session.OnAllComplete = func() {
		session.Destroy()
		if onDone != nil {
			onDone()
		}
	}
	b.turnSession = session
	session.Start()
}

// ── Dialogue box scheduling ──────────────────────────────────────

func (b *Battle) ScheduleDialogueBox(id *interface{}) {
	shouldShow := b.State == StateEnemyTarget && b.DialogueBoxIcon != nil

	if shouldShow && *id == nil {
		d := &dialogueBoxDrawer{battle: b}
		queues.DefaultQueue.ScheduleAt(d, 150)
		*id = d
	} else if !shouldShow && *id != nil {
		queues.DefaultQueue.Unschedule((*id).(queues.Drawable))
		*id = nil
	}
}

type dialogueBoxDrawer struct {
	battle *Battle
}

func (d *dialogueBoxDrawer) Draw(s *ebiten.Image) {
	d.battle.drawDialogueBox(s)
}

// ── Narrative helpers ────────────────────────────────────────────

func (b *Battle) clearNarrativeText() {
	if b.turnSession != nil {
		b.turnSession.Destroy()
		b.turnSession = nil
	}
	if b.restoredText != nil {
		b.restoredText.Destroy()
		b.restoredText = nil
	}
}

func (b *Battle) restoreNarrative() {
	if len(b.narrativeLines) == 0 {
		return
	}
	if b.restoredText != nil {
		b.restoredText.Destroy()
		b.restoredText = nil
	}

	lastLine := b.narrativeLines[len(b.narrativeLines)-1]
	td, _ := b.TextEngine.DisplayText(
		engine.StyleNarrative.WithInstant(true),
		lastLine,
		b.SoundPlayer,
		nil,
	)
	b.restoredText = td
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

// ── Character animations ─────────────────────────────────────────

func (b *Battle) updateCharacterAnimations(dt time.Duration) {
	for _, m := range b.Party {
		if m.CharacterSprite == nil {
			continue
		}
		if m.PlayingDefendRev {
			m.DefendRevFrame--
			if m.DefendRevFrame < 0 {
				m.PlayingDefendRev = false
				m.CharacterSprite.SetIconState("idle")
			} else {
				m.CharacterSprite.SetIconState("defend")
				m.CharacterSprite.CurrentState.CurrentFrame = m.DefendRevFrame
			}
		} else if m.IsDefending {
			cs := m.CharacterSprite
			if cs.CurrentState.Name != "defend" {
				cs.SetIconState("defend")
				cs.CurrentState.Mode = render.AnimationModeOnce
			}
			cs.Update(dt)
		} else {
			cs := m.CharacterSprite
			if cs.CurrentState.Name != "idle" {
				cs.SetIconState("idle")
				cs.CurrentState.Mode = render.AnimationModeLoop
			}
			cs.Update(dt)
		}
	}
	for _, o := range b.Opponents {
		if o.CharacterSprite != nil {
			o.CharacterSprite.Update(dt)
		}
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

	case b.State.IsExecuting():
		// narrative auto-advances via queue
		return

	case b.State == StateEnemyTarget:
		b.updateTurnWait(dt)
		return

	case b.State == StateEnemyTurn:
		b.updateSoulMovement(dt)
		b.updateTurnWait(dt)
		b.updateAttack(dt)
		return
	}
}

func (b *Battle) updateSoulMovement(dt time.Duration) {
	soulSpeed := 200.0
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		b.SoulX -= soulSpeed * dt.Seconds()
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		b.SoulX += soulSpeed * dt.Seconds()
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		b.SoulY -= soulSpeed * dt.Seconds()
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		b.SoulY += soulSpeed * dt.Seconds()
	}
	if b.ArenaBoundsW > 0 && b.ArenaBoundsH > 0 {
		if b.SoulX < b.ArenaBoundsX {
			b.SoulX = b.ArenaBoundsX
		}
		if b.SoulX > b.ArenaBoundsX+b.ArenaBoundsW {
			b.SoulX = b.ArenaBoundsX + b.ArenaBoundsW
		}
		if b.SoulY < b.ArenaBoundsY {
			b.SoulY = b.ArenaBoundsY
		}
		if b.SoulY > b.ArenaBoundsY+b.ArenaBoundsH {
			b.SoulY = b.ArenaBoundsY + b.ArenaBoundsH
		}
	}
	if b.SoulCollider != nil {
		t := &components.Transform{X: b.SoulX, Y: b.SoulY, ScaleX: 2, ScaleY: 2}
		b.SoulCollider.UpdateWorldVerts(t)
	}
}

func (b *Battle) updateTurnWait(dt time.Duration) {
	if b.turnWaitingForZ {
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			b.turnWaitingForZ = false
			cb := b.turnWaitCallback
			b.turnWaitCallback = nil
			if cb != nil {
				cb()
			}
		}
		return
	}

	if b.turnWaitingForTimer {
		b.turnTimerElapsed += dt
		if b.turnTimerElapsed >= b.turnTimerTarget {
			b.turnWaitingForTimer = false
			cb := b.turnWaitCallback
			b.turnWaitCallback = nil
			if cb != nil {
				cb()
			}
		}
		return
	}
}

func (b *Battle) updateAttack(dt time.Duration) {
	if b.turnAttackSeq == nil {
		return
	}

	b.turnAttackSeq.Update(dt)
	b.turnAttackElapsed += dt

	if b.SoulCollider != nil && len(b.Targets) > 0 {
		alive := b.turnAttackSeq.ActiveProjectiles[:0]
		for _, p := range b.turnAttackSeq.ActiveProjectiles {
			if p.Collider != nil && p.Transform != nil && b.SoulCollider.CollidesWith(p.Collider) {
				for _, tIdx := range b.Targets {
					if tIdx >= 0 && tIdx < len(b.Party) && b.Party[tIdx].Alive() {
						b.Party[tIdx].TakeDamage(b, p.Damage)
						break
					}
				}
				b.retargetNext()
				queues.QDel(p)
				continue
			}
			alive = append(alive, p)
		}
		b.turnAttackSeq.ActiveProjectiles = alive
	}

	if b.turnAttackElapsed >= b.turnAttackDuration {
		b.turnAttackSeq = nil
		cb := b.turnAttackDone
		b.turnAttackDone = nil
		if cb != nil {
			cb()
		}
	}
}

// ── Target selection ─────────────────────────────────────────────

func (b *Battle) selectTargets() {
	var alive []int
	for i, m := range b.Party {
		if m.Alive() {
			alive = append(alive, i)
		}
	}

	b.Targets = nil
	if len(alive) == 0 {
		return
	}

	numTargets := 2
	if len(alive) < 2 {
		numTargets = len(alive)
	} else {
		maxExtra := len(alive) - 2
		if maxExtra > 1 {
			maxExtra = 1
		}
		numTargets = 2 + rand.Intn(maxExtra+1)
	}

	rand.Shuffle(len(alive), func(i, j int) {
		alive[i], alive[j] = alive[j], alive[i]
	})

	b.Targets = make([]int, numTargets)
	copy(b.Targets, alive[:numTargets])
}

func (b *Battle) retargetNext() {
	if len(b.Targets) == 0 {
		return
	}

	for i, tIdx := range b.Targets {
		if tIdx >= 0 && tIdx < len(b.Party) && b.Party[tIdx].Alive() {
			continue
		}
		for j, m := range b.Party {
			if !m.Alive() {
				continue
			}
			already := false
			for _, t := range b.Targets {
				if t == j {
					already = true
					break
				}
			}
			if !already {
				b.Targets[i] = j
				break
			}
		}
	}
}
