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
	menuY       = 668.0
	memberInfoY = 820.0

	member1X = 40.0
	member2X = 0.0
	member3X = 45
)

// PillarParticle is a vertical bar spawned from card pillar tips,
// drifting inward and fading out. Coloured with the card's accent.
type PillarParticle struct {
	X, Y      float64
	Dir       float64 // -1 = left, +1 = right
	Accent    color.RGBA
	SpawnAt   time.Time
	CardIndex int // which party member this particle tracks
}

// CommittedAction records what a party member chose to do this turn.
type CommittedAction struct {
	ActionType   int    // BtnFight, BtnActMagic, etc.
	ActName      string // populated for ACTs
	TargetIdx    int    // index into Party or Opponents (-1 if none)
	IsAllyTarget bool   // true = Party member, false = Opponent
}

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
	SelectedTarget int  // index into the target list when MenuTarget
	targetIsAlly   bool // true = targeting Party, false = targeting Opponents
	PendingActName string

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
	actedCount       int                // how many party members have committed their action this turn
	CommittedActions []*CommittedAction // per-member committed action (indexed by Party position)

	// ── Card animation ──
	cardAnimY []float64 // current Y per card, lerps toward target

	// ── Pillar particles ──
	pillarParticles   []PillarParticle
	lastParticleSpawn time.Time

	// Arena layer visibility hooks — set by game.go
	showArena    func()
	hideArena    func()
	ArenaBoundsX float64 // inner area where the soul can move
	ArenaBoundsY float64
	ArenaBoundsW float64
	ArenaBoundsH float64

	// ── Soul (heart) position during enemy turn ──
	SoulX float64
	SoulY float64

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

// SetArenaBounds sets the inner play area where the soul can move.
func (b *Battle) SetArenaBounds(x, y, w, h float64) {
	b.ArenaBoundsX = x
	b.ArenaBoundsY = y
	b.ArenaBoundsW = w
	b.ArenaBoundsH = h
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
	case MenuTarget:
		b.navigateTargets()
	}
}

func (b *Battle) navigateMain() {
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		b.SelectedButton = (b.SelectedButton - 1 + BtnCount) % BtnCount
		b.SoundPlayer.PlaySound("squeak", 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		b.SelectedButton = (b.SelectedButton + 1) % BtnCount
		b.SoundPlayer.PlaySound("squeak", 1)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		b.SoundPlayer.PlaySound("select", 1)
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
		b.SoundPlayer.PlaySound("squeak", 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		b.SelectedAct = (b.SelectedAct + 1) % len(acts)
		b.SoundPlayer.PlaySound("squeak", 1)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.MenuState = MenuMain
		// Restore narrator text instantly
		b.restoreNarrative()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		actDef := acts[b.SelectedAct].Def
		engine.ShowDebugNotice(b.TextEngine, "act selected: "+actDef.Name, 10, 10, 2*time.Second)
		b.SoundPlayer.PlaySound("select", 1)
		if actDef.TargetSelf {
			// Target a party member — show target selection
			b.PendingActName = actDef.Name
			b.SelectedTarget = 0
			b.targetIsAlly = true
			b.MenuState = MenuTarget
			b.clearNarrativeText()
		} else if len(b.Opponents) > 1 {
			// Multiple opponents — show target selection
			b.PendingActName = actDef.Name
			b.SelectedTarget = 0
			b.targetIsAlly = false
			b.MenuState = MenuTarget
			b.clearNarrativeText()
		} else {
			// Single opponent — auto-target
			b.commitAction(BtnActMagic, actDef.Name, 0, false)
			engine.ShowDebugNotice(b.TextEngine, "committed: "+actDef.Name, 10, 30, 2*time.Second)
			b.advanceToNextMember()
		}
	}
}

func (b *Battle) navigateTargets() {
	var targetCount int
	if b.targetIsAlly {
		targetCount = len(b.Party)
	} else {
		targetCount = len(b.Opponents)
	}
	if targetCount == 0 {
		b.MenuState = MenuAct
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		b.SelectedTarget = (b.SelectedTarget - 1 + targetCount) % targetCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		b.SelectedTarget = (b.SelectedTarget + 1) % targetCount
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.MenuState = MenuAct
		b.restoreNarrative()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		b.commitAction(BtnActMagic, b.PendingActName, b.SelectedTarget, b.targetIsAlly)
		b.PendingActName = ""
		engine.ShowDebugNotice(b.TextEngine, "committed: "+b.CommittedActions[b.ActiveMember].ActName, 10, 30, 2*time.Second)
		b.advanceToNextMember()
	}
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

	// Draw soul inside the arena during the enemy's attack phase
	if b.State == StateTurnPlaying && b.SoulSprite != nil {
		b.SoulSprite.Draw(screen, b.SoulX, b.SoulY, 2.0, 2.0, 0)
	}

	if b.State != StatePlayerAction || b.MenuSprite == nil {
		return
	}

	debugStr := fmt.Sprintf("Hero: %d/%d", b.ActiveMember+1, len(b.Party))
	ebitenutil.DebugPrintAt(screen, debugStr, 4, 4)

	b.drawMainButtons(screen)

	if b.MenuState == MenuAct {
		b.drawActList(screen)
	}
	if b.MenuState == MenuTarget {
		b.drawTargetList(screen)
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
	baseDownY := 657.0
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
		FontName: "roarin", ScaleX: 0.6, ScaleY: 0.3,
		FontHeight: 24.0, LineSpacing: 0, DefaultDelay: 0.03,
		CharSpacing: -10,
	}

	for i, m := range b.Party {
		isFocused := i == b.ActiveMember && b.State == StatePlayerAction
		targetY := baseDownY
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

		if isFocused {
			ebitenutil.DrawRect(screen, cx, cardY-borderW,
				cardW, borderW, accent)
			ebitenutil.DrawRect(screen, cx, cardY+cardH,
				cardW, borderW, accent)
			ebitenutil.DrawRect(screen, cx-borderW, cardY-borderW,
				borderW, cardH+borderW*2+pillarH, accent)
			ebitenutil.DrawRect(screen, cx+cardW, cardY-borderW,
				borderW, cardH+borderW*2+pillarH, accent)
			ebitenutil.DrawRect(screen, cx, cardY, cardW, cardH,
				color.RGBA{0, 0, 0, 240})
		}

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

		hpStr := fmt.Sprintf("%d / %d", int(m.HP), int(m.MaxHP))
		b.drawMenuString(screen, hpStyle, hpStr, cx+245, cardY-10)

		if m.BattleMiniature != nil {
			m.BattleMiniature.Draw(screen, cx, cardY-5, 2, 2, 0)
		}
	}
	// ── Pillar particles: spawn & cleanup ──
	now := time.Now()
	const spawnInterval = 500 * time.Millisecond
	// Clear particles when the tracked card is no longer focused.
	if len(b.pillarParticles) > 0 {
		idx := b.pillarParticles[0].CardIndex
		if idx != b.ActiveMember || b.State != StatePlayerAction {
			b.pillarParticles = b.pillarParticles[:0]
		}
	}
	if b.State == StatePlayerAction {
		// Sync particle Y with current card animation position.
		for i := range b.pillarParticles {
			p := &b.pillarParticles[i]
			if p.CardIndex < len(b.cardAnimY) {
				// tipY = cardY + cardH + borderW + pillarH  (constants below)
				p.Y = b.cardAnimY[p.CardIndex] + 70 + 4 + 70
			}
		}
		b.drawPillarParticles(screen, now)

		if now.Sub(b.lastParticleSpawn) >= spawnInterval {
			b.lastParticleSpawn = now

			cardW := 424.0
			cardH := 70.0
			cardGap := 4.0
			startX := 4.0
			borderW := 4.0
			pillarH := 70.0

			for i := range b.Party {
				if i != b.ActiveMember || b.State != StatePlayerAction {
					continue
				}
				cardY := 650.0
				if b.State == StatePlayerAction {
					cardY = 585.0
				}
				if i < len(b.cardAnimY) {
					cardY = b.cardAnimY[i]
				}
				cx := startX + float64(i)*(cardW+cardGap)
				tipY := cardY + cardH + borderW + pillarH

				accent, _ := b.Party[b.ActiveMember].AccentColor.(color.RGBA)
				b.pillarParticles = append(b.pillarParticles,
					PillarParticle{X: cx - borderW, Y: tipY, Dir: 1, Accent: accent, SpawnAt: now, CardIndex: b.ActiveMember},
					PillarParticle{X: cx + cardW, Y: tipY, Dir: -1, Accent: accent, SpawnAt: now, CardIndex: b.ActiveMember},
				)
				break
			}
		}

		// Expire particles older than 1 second.
		cutoff := now.Add(-1 * time.Second)
		alive := b.pillarParticles[:0]
		for _, p := range b.pillarParticles {
			if p.SpawnAt.After(cutoff) {
				alive = append(alive, p)
			}
		}
		b.pillarParticles = alive
	} // end if (StatePlayerAction)
}

func (b *Battle) drawPillarParticles(screen *ebiten.Image, now time.Time) {
	barW := 4.0
	barH := 70.0

	for _, p := range b.pillarParticles {
		age := now.Sub(p.SpawnAt).Seconds()
		if age > 1.0 {
			continue
		}
		x := p.X + p.Dir*30*age // drift inward
		y := p.Y - barH         // bar sits above the pillar tip
		alpha := uint8(255 * (1.0 - age))
		a := float32(alpha) / 255
		clr := color.RGBA{
			R: uint8(float32(p.Accent.R) * a),
			G: uint8(float32(p.Accent.G) * a),
			B: uint8(float32(p.Accent.B) * a),
			A: alpha,
		}
		ebitenutil.DrawRect(screen, x, y, barW, barH, clr)
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

func (b *Battle) drawTargetList(screen *ebiten.Image) {
	startY := menuY + menuBtnH
	lineH := 60.0

	var names []string
	if b.targetIsAlly {
		for _, m := range b.Party {
			names = append(names, strings.ToUpper(m.Name))
		}
	} else {
		for _, o := range b.Opponents {
			names = append(names, strings.ToUpper(o.Name))
		}
	}

	if len(names) == 0 {
		return
	}

	// Soul cursor
	if b.SoulSprite != nil {
		soulX := 60.0
		soulY := startY + float64(b.SelectedTarget)*lineH + 8
		b.SoulSprite.Draw(screen, soulX, soulY, 2.0, 2.0, 0)
	}

	listStartX := 124.0
	targetStyle := engine.TextStyle{
		FontName:     "determination",
		ScaleX:       0.5,
		ScaleY:       0.5,
		FontHeight:   24.0,
		LineSpacing:  60,
		DefaultDelay: 0.03,
	}

	for i, name := range names {
		y := startY + float64(i)*lineH
		b.drawMenuString(screen, targetStyle, name, listStartX, y-25)
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

// commitAction saves the current member's chosen action for execution at turn end.
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
	if b.ActiveMember >= 0 && b.ActiveMember < len(b.CommittedActions) {
		b.CommittedActions[b.ActiveMember] = nil
	}
	b.SelectedButton = 0
	b.MenuState = MenuMain
	b.restoreNarrative()
	if b.SoundPlayer != nil {
		b.SoundPlayer.PlaySound("smallswing", 1.0)
	}
}

func (b *Battle) StartTurn(turn *Turn, narrativeLines []string) {
	b.State = StatePlayerAction
	b.MenuState = MenuMain
	b.SelectedButton = 0
	b.SelectedAct = 0
	b.SelectedTarget = 0
	b.targetIsAlly = false
	b.PendingActName = ""
	b.actedCount = 0
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
}

func (b *Battle) FinishPlayerTurn() {
	b.State = StateTurnPlaying
	b.MenuState = MenuHidden
	b.clearNarrativeText()
	// Place the soul at the centre of the arena
	if b.ArenaBoundsW > 0 && b.ArenaBoundsH > 0 {
		b.SoulX = b.ArenaBoundsX + b.ArenaBoundsW/2
		b.SoulY = b.ArenaBoundsY + b.ArenaBoundsH/2
	}
	if b.showArena != nil {
		b.showArena()
	}
	b.turnPlayer.Start()
}

func (b *Battle) Update(dt time.Duration) {
	if b.State != StateTurnPlaying {
		return
	}

	// Soul movement during enemy turn
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
	// Constrain soul within arena bounds
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
