package battle

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
)

// ── Top-level states ─────────────────────────────────────────────

type BattleState int

const (
	StateIdle BattleState = iota
	StatePlayerAction
	StateTurnPlaying
)

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

// ── Button positioning ───────────────────────────────────────────

const (
	menuScale   = 3.0
	menuBtnW    = 20.0 * menuScale
	menuBtnH    = 32.0 * menuScale
	menuBtnGap  = 10.0
	menuY       = 665.0
	memberInfoY = 820.0

	member1X = 40.0
	member2X = 0.0
	member3X = 45
)

// ── Battle ───────────────────────────────────────────────────────

type Battle struct {
	State       BattleState
	TextEngine  *engine.TextEngine
	SoundPlayer *sound.SoundPlayer

	// ── Encounter data ──
	Party        []*PartyMember
	Opponents    []*Opponent
	ActiveMember int

	// ── Menu state ──
	MenuState      BattleMenuState
	SelectedButton int
	SelectedAct    int

	// ── Menu sprite ──
	MenuSprite *render.AnimatedIcon
	SoulSprite *render.AnimatedIcon

	// ── Turn events state ──
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

	currentCharacterSelecting int

	// Saved narrative: used to restore narrator text when returning from sub-menus
	narrativeLines []string
	restoredText   *engine.TextDisplay

	// ── Party turn state ──
	actedCount int // how many party members have committed their action this turn

	// ── Card animation ──
	cardAnimY []float64 // current Y per card, lerps toward target

	// Arena layer visibility hooks — set by game.go
	showArena func()
	hideArena func()

	OnTurnComplete func()
}

// SetMenuSprite sets the AnimatedIcon used for rendering buttons.
func (b *Battle) SetMenuSprite(s *render.AnimatedIcon) {
	b.MenuSprite = s
}

// SetSoulSprite sets the AnimatedIcon used for the soul cursor.
func (b *Battle) SetSoulSprite(s *render.AnimatedIcon) {
	b.SoulSprite = s
}

// SetArenaHooks registers callbacks to show/hide arena layers.
func (b *Battle) SetArenaHooks(show, hide func()) {
	b.showArena = show
	b.hideArena = hide
}

// ── Button label helpers ─────────────────────────────────────────

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

// ── Menu navigation ──────────────────────────────────────────────

func (b *Battle) NavigateMenu() {
	if b.State != StatePlayerAction {
		return
	}

	switch b.MenuState {
	case MenuMain:
		b.navigateMain()
	case MenuAct:
		b.navigateActs()
	}
}

func (b *Battle) navigateMain() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		b.SelectedButton = (b.SelectedButton - 1 + BtnCount) % BtnCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		b.SelectedButton = (b.SelectedButton + 1) % BtnCount
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		switch b.SelectedButton {
		case BtnFight:
			// TODO: resolve FIGHT
		case BtnActMagic:
			b.MenuState = MenuAct
			b.SelectedAct = 0
			// Hide narrator text so it doesn't overlap the ACT menu
			b.clearNarrativeText()
		case BtnItem:
			// TODO: open inventory
		case BtnSpare:
			// TODO: resolve SPARE
		case BtnDefend:
			if m := b.Party[b.ActiveMember]; m != nil {
				m.IsDefending = true
			}
			b.advanceToNextMember()
		}
		return
	}

	// X — undo previous member's action and return focus to them
	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.undoLastMember()
	}
}

func (b *Battle) navigateActs() {
	acts := b.CollectActs()
	if len(acts) == 0 {
		b.MenuState = MenuMain
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		b.SelectedAct = (b.SelectedAct - 1 + len(acts)) % len(acts)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		b.SelectedAct = (b.SelectedAct + 1) % len(acts)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.MenuState = MenuMain
		// Restore narrator text instantly
		b.restoreNarrative()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		actName := acts[b.SelectedAct].Def.Name
		engine.ShowDebugNotice(b.TextEngine, "act selected: "+actName, 10, 10, 2*time.Second)
		// TODO: resolve ACT effect
		b.advanceToNextMember()
	}
}

func (b *Battle) CollectActs() []ActEntry {
	var leader, active *PartyMember
	if len(b.Party) > 0 {
		active = b.Party[b.ActiveMember]
		for _, m := range b.Party {
			if m.IsLeader {
				leader = m
				break
			}
		}
	}
	var target *Opponent
	if len(b.Opponents) > 0 {
		target = b.Opponents[0]
	}
	return CollectActs(leader, active, target)
}

func (b *Battle) ActiveMemberSwitchedTo(idx int) {
	if idx < 0 || idx >= len(b.Party) {
		return
	}
	b.ActiveMember = idx
	b.SelectedButton = 0
	b.MenuState = MenuMain
}

// ── Menu drawing ─────────────────────────────────────────────────

// drawMenuString renders static text at (x, y) using TextDisplay instant mode.
// Uses the font and scale from the given style, overriding StartX/StartY.
func (b *Battle) drawMenuString(screen *ebiten.Image, style engine.TextStyle, text string, x, y float64) {
	// Lazy-load font if not yet cached (same logic as TextEngine.DisplayText).
	if _, exists := b.TextEngine.FontsLoaded[style.FontName]; !exists {
		icon, err := render.NewAnimatedIconFromPath("media/sprites/"+style.FontName, " ")
		if err != nil {
			return
		}
		b.TextEngine.FontsLoaded[style.FontName] = *icon
	}
	font := b.TextEngine.FontsLoaded[style.FontName]

	td := &engine.TextDisplay{
		Font:        font,
		Text:        text,
		StartX:      x,
		StartY:      y,
		ScaleX:      style.ScaleX,
		ScaleY:      style.ScaleY,
		FontHeight:  style.FontHeight,
		LineSpacing: style.LineSpacing,
		Delay:       style.DefaultDelay,
		Instant:     true,
		CharWidth:   make(map[string]int),
		CharSpacing: style.CharSpacing,
		Displayed:   make([]*engine.Glyph, 0),
	}
	td.Parse()
	td.Update(0) // build all glyphs immediately via Instant mode
	td.Draw(screen)
	td.Destroy()
}

// calcButtonsWidth returns the total width of the main button row.
func (b *Battle) calcButtonsWidth() float64 {
	return float64(BtnCount)*menuBtnW + float64(BtnCount-1)*menuBtnGap
}

// DrawMenu renders the battle menu and party info on the given screen.
func (b *Battle) DrawMenu(screen *ebiten.Image) {
	// Cards always visible — slide down during enemy turn.
	b.drawMemberCards(screen)

	if b.State != StatePlayerAction || b.MenuSprite == nil {
		return
	}

	debugStr := fmt.Sprintf("Hero: %d/%d", b.ActiveMember+1, len(b.Party))
	ebitenutil.DebugPrintAt(screen, debugStr, 4, 4)

	b.drawMainButtons(screen)

	if b.MenuState == MenuAct {
		b.drawActList(screen)
	}
}

func (b *Battle) drawMemberCards(screen *ebiten.Image) {
	if len(b.Party) == 0 {
		return
	}

	cardW := 424.0
	cardH := 70.0
	cardGap := 4.0
	startX := 4.0
	baseDownY := 650.0
	baseUpY := 585.0
	borderW := 4.0
	pillarH := 70.0 // extra height below card for focused "legs"
	const slideSpeed = 0.15

	if len(b.cardAnimY) != len(b.Party) {
		b.cardAnimY = make([]float64, len(b.Party))
		for i := range b.cardAnimY {
			b.cardAnimY[i] = baseDownY
		}
	}

	nameStyle := engine.TextStyle{
		FontName: "roarin", ScaleX: 0.6, ScaleY: 0.7,
		FontHeight: 24.0, LineSpacing: 0, DefaultDelay: 0.03,
		CharSpacing: -9,
	}

	hpStyle := engine.TextStyle{
		FontName: "cryptoftomorrow", ScaleX: 0.3, ScaleY: 0.3,
		FontHeight: 24.0, LineSpacing: 0, DefaultDelay: 0.03,
		CharSpacing: 0,
	}

	for i, m := range b.Party {
		targetY := baseDownY
		isFocused := i == b.ActiveMember && b.State == StatePlayerAction
		if isFocused {
			targetY = baseUpY
		}
		b.cardAnimY[i] += (targetY - b.cardAnimY[i]) * slideSpeed
		cardY := b.cardAnimY[i]

		cx := startX + float64(i)*(cardW+cardGap)

		accent, ok := m.AccentColor.(color.RGBA)
		if !ok {
			accent = color.RGBA{128, 128, 128, 255}
		}
		dimAccent := color.RGBA{accent.R / 3, accent.G / 3, accent.B / 3, accent.A}

		if isFocused {
			// ── Focused: full border with pillars extending below ──
			// Top bar
			ebitenutil.DrawRect(screen, cx, cardY-borderW,
				cardW, borderW, accent)
			// Bottom bar
			ebitenutil.DrawRect(screen, cx, cardY+cardH,
				cardW, borderW, accent)
			// Left pillar (extends below card)
			ebitenutil.DrawRect(screen, cx-borderW, cardY-borderW,
				borderW, cardH+borderW*2+pillarH, accent)
			// Right pillar (extends below card)
			ebitenutil.DrawRect(screen, cx+cardW, cardY-borderW,
				borderW, cardH+borderW*2+pillarH, accent)
		} else {
			// ── Unfocused: top and bottom bars only, dimmed ──
			ebitenutil.DrawRect(screen, cx, cardY-borderW,
				cardW, borderW, dimAccent)
			ebitenutil.DrawRect(screen, cx, cardY+cardH,
				cardW, borderW, dimAccent)
		}

		// Card background
		ebitenutil.DrawRect(screen, cx, cardY, cardW, cardH,
			color.RGBA{0, 0, 0, 240})

		hpBarX := cx + 250
		hpBarY := cardY + cardH - 28
		hpBarW := 155.0
		hpBarH := 18.0

		ebitenutil.DrawRect(screen, hpBarX, hpBarY, hpBarW, hpBarH,
			color.RGBA{40, 40, 40, 255})
		if m.MaxHP > 0 {
			fill := hpBarW * (m.HP / m.MaxHP)
			if fill < 1 && m.HP > 0 {
				fill = 1
			}
			hpClr := color.RGBA{255, 220, 0, 255}
			if m.HP < m.MaxHP*0.25 && isFocused {
				hpClr = color.RGBA{255, 60, 40, 255}
			}
			ebitenutil.DrawRect(screen, hpBarX, hpBarY, fill, hpBarH, hpClr)
		}

		b.drawMenuString(screen, nameStyle, strings.ToUpper(m.Name), cx+99, cardY-48)

		hpStr := fmt.Sprintf("%d/%d", int(m.HP), int(m.MaxHP))
		b.drawMenuString(screen, hpStyle, hpStr, cx+250, cardY-10)

		if m.BattleMiniature != nil {
			m.BattleMiniature.Draw(screen, cx, cardY, 2, 2, 0)
		}
	}
}

func (b *Battle) drawMainButtons(screen *ebiten.Image) {
	totalW := b.calcButtonsWidth()
	screenW := float64(screen.Bounds().Dx())

	// Reposition buttons based on active party member
	var startX float64
	switch b.ActiveMember {
	case 0:
		startX = member1X
	case 1:
		startX = (screenW-totalW)/2 + member2X
	case 2:
		startX = screenW - totalW - member3X
	default:
		startX = (screenW - totalW) / 2
	}

	for i := 0; i < BtnCount; i++ {
		x := startX + float64(i)*(menuBtnW+menuBtnGap)

		inactiveName := b.buttonSpriteNameFallback(i)
		b.MenuSprite.SetIconState(inactiveName)

		if i == b.SelectedButton {
			activeName := b.buttonSpriteName(i)
			b.MenuSprite.SetIconState(activeName)
		}

		b.MenuSprite.Draw(screen, x, menuY, 2, 2, 0)
	}
}

// buttonSpriteName returns the menu sprite tag for a given button slot.
// Selected button gets the _active variant; label (ACT/MAGIC) determines which sprite.
func (b *Battle) buttonSpriteName(btn int) string {
	active := btn == b.SelectedButton

	switch btn {
	case BtnFight:
		if active {
			return "fight_active"
		}
		return "fight"
	case BtnActMagic:
		if b.ActMagicLabel() == "ACT" {
			if active {
				return "act_active"
			}
			return "act"
		}
		if active {
			return "magic_active"
		}
		return "magic"
	case BtnItem:
		if active {
			return "item_active"
		}
		return "item"
	case BtnSpare:
		if active {
			return "spare_active"
		}
		return "spare"
	case BtnDefend:
		if active {
			return "defend_active"
		}
		return "defend"
	}
	return "fight"
}

// buttonSpriteNameFallback returns the inactive sprite name for a button.
func (b *Battle) buttonSpriteNameFallback(btn int) string {
	switch btn {
	case BtnFight:
		return "fight"
	case BtnActMagic:
		if b.ActMagicLabel() == "ACT" {
			return "act"
		}
		return "magic"
	case BtnItem:
		return "item"
	case BtnSpare:
		return "spare"
	case BtnDefend:
		return "defend"
	}
	return "fight"
}

func (b *Battle) drawActList(screen *ebiten.Image) {
	acts := b.CollectActs()
	if len(acts) == 0 {
		return
	}

	screenW := float64(screen.Bounds().Dx())

	// ── Layout: ACT list at the bottom, below the main buttons ──
	actStartY := menuY + menuBtnH // below the button row
	actLineH := 60.0

	// ── Soul sprite cursor ──
	if b.SoulSprite != nil {
		soulX := 60.0
		soulY := actStartY + float64(b.SelectedAct)*actLineH + 8
		b.SoulSprite.Draw(screen, soulX, soulY, 2.0, 2.0, 0)
	}

	// ── Act names ──
	listStartX := 124.0

	actStyle := engine.TextStyle{
		FontName:     "determination",
		ScaleX:       0.5,
		ScaleY:       0.5,
		FontHeight:   24.0,
		LineSpacing:  60,
		DefaultDelay: 0.03,
	}

	for i, entry := range acts {
		y := actStartY + float64(i)*actLineH
		b.drawMenuString(screen, actStyle, entry.Def.Name, listStartX, y-25)
	}

	// ── Description of the selected act (right side, same vertical band) ──
	if b.SelectedAct >= 0 && b.SelectedAct < len(acts) {
		desc := acts[b.SelectedAct].Def.Description
		if desc != "" {
			descX := screenW - 340.0
			descY := actStartY - 8

			descStyle := engine.TextStyle{
				FontName:     "determination",
				ScaleX:       0.5,
				ScaleY:       0.5,
				FontHeight:   24.0,
				LineSpacing:  6,
				DefaultDelay: 0.03,
			}

			lines := wrapText(desc, 42)
			for li, line := range lines {
				b.drawMenuString(screen, descStyle, strings.TrimSpace(line), descX, descY+float64(li)*20)
			}
		}
	}
}

// wrapText splits text into lines, each at most maxChars runes.
func wrapText(text string, maxChars int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len([]rune(cur))+1+len([]rune(w)) > maxChars {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	lines = append(lines, cur)
	return lines
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
		tp.battle.State = StateIdle
		if tp.battle.OnTurnComplete != nil {
			tp.battle.OnTurnComplete()
		}
		return
	}

	event := tp.turn.Sequence[tp.index]
	tp.index++
	event.Run(tp.battle, func() { tp.next() })
}

// ── Battle lifecycle ─────────────────────────────────────────────

// clearNarrativeText destroys both the original session and any restored text display.
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

// restoreNarrative re-renders the last narrative line instantly.
func (b *Battle) restoreNarrative() {
	if len(b.narrativeLines) == 0 {
		return
	}
	// Clear any previously restored text first
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

// advanceToNextMember locks in the current member's action and moves to the next.
// If all members have acted, the battle turn begins.
func (b *Battle) advanceToNextMember() {
	b.actedCount++
	if b.actedCount >= len(b.Party) {
		b.FinishPlayerTurn()
		return
	}
	b.ActiveMember = b.actedCount
	b.SelectedButton = 0
	b.MenuState = MenuMain
	b.restoreNarrative()
}

// undoLastMember reverts the most recently committed member's action
// and returns focus to them.
func (b *Battle) undoLastMember() {
	if b.actedCount <= 0 {
		return
	}
	b.actedCount--
	b.ActiveMember = b.actedCount
	b.SelectedButton = 0
	b.MenuState = MenuMain
	b.restoreNarrative()
}

func (b *Battle) StartTurn(turn *Turn, narrativeLines []string) {
	b.State = StatePlayerAction
	b.MenuState = MenuHidden
	b.SelectedButton = 0
	b.SelectedAct = 0
	b.actedCount = 0
	b.turnPlayer = NewTurnPlayer(b, turn)
	b.narrativeLines = narrativeLines

	if len(narrativeLines) > 0 {
		session := engine.NewDialogueSession(
			b.TextEngine, narrativeLines, engine.StyleNarrative, b.SoundPlayer,
		)
		session.OnAllComplete = func() {
			b.MenuState = MenuMain
		}
		b.turnSession = session
		session.Start()
	} else {
		b.MenuState = MenuMain
	}
}

func (b *Battle) FinishPlayerTurn() {
	b.State = StateTurnPlaying
	b.MenuState = MenuHidden
	if b.showArena != nil {
		b.showArena()
	}
	b.turnPlayer.Start()
}

func (b *Battle) Update(dt time.Duration) {
	if b.State != StateTurnPlaying {
		return
	}

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

	if b.turnAttackSeq != nil {
		b.turnAttackSeq.Update(dt)
		b.turnAttackElapsed += dt
		if b.turnAttackElapsed >= b.turnAttackDuration {
			b.turnAttackSeq = nil
			cb := b.turnAttackDone
			b.turnAttackDone = nil
			if cb != nil {
				cb()
			}
		}
		return
	}
}

func (b *Battle) ShowActionMenus() {
	// TODO: implement
}
