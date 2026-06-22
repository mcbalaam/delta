package battle

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
)

// ── Menu navigation ──────────────────────────────────────────────

func (b *Battle) NavigateMenu() {
	if !b.State.IsSelecting() {
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
			b.MenuState = MenuTarget
			b.SelectedTarget = 0
			b.targetIsAlly = false
			b.clearNarrativeText()
		case BtnItem:
			// TODO: open inventory
		case BtnSpare:
			// TODO: resolve SPARE
		case BtnDefend:
			if m := b.Party[b.ActiveMember]; m != nil {
				m.IsDefending = true
			}
			b.advanceFromSelecting()
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.undoLastMember()
	}
}

func (b *Battle) navigateActs() {
	acts := b.CollectActs()
	if len(acts) == 0 {
		b.MenuState = MenuTarget
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
		b.MenuState = MenuTarget
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		b.SoundPlayer.PlaySound("select", 1)
		actDef := acts[b.SelectedAct].Def
		b.commitAction(BtnActMagic, actDef.Name, b.SelectedTarget, b.targetIsAlly)
		b.advanceFromSelecting()
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
		b.MenuState = MenuMain
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		b.SelectedTarget = (b.SelectedTarget - 1 + targetCount) % targetCount
		b.SoundPlayer.PlaySound("squeak", 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		b.SelectedTarget = (b.SelectedTarget + 1) % targetCount
		b.SoundPlayer.PlaySound("squeak", 1)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		b.MenuState = MenuMain
		b.restoreNarrative()
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		b.SoundPlayer.PlaySound("select", 1)
		b.MenuState = MenuAct
		b.SelectedAct = 0
	}
}

// ── Menu drawing ─────────────────────────────────────────────────

func (b *Battle) drawMenuString(screen *ebiten.Image, style engine.TextStyle, text string, x, y float64) {
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
	td.Update(0)
	td.Draw(screen)
	td.Destroy()
}

func (b *Battle) calcButtonsWidth() float64 {
	return float64(BtnCount)*menuBtnW + float64(BtnCount-1)*menuBtnGap
}

func (b *Battle) DrawMenu(screen *ebiten.Image) {
	b.drawMemberCards(screen)
	b.drawPartyOnArena(screen)
	b.drawOpponents(screen)

	if b.State == StateEnemyTurn && b.SoulSprite != nil {
		b.SoulSprite.Draw(screen, b.SoulX, b.SoulY, 2.0, 2.0, 0)
	}

	b.drawTargetIcons(screen)

	if !b.State.IsSelecting() || b.MenuSprite == nil {
		return
	}

	b.drawMainButtons(screen)

	if b.MenuState == MenuAct {
		b.drawActList(screen)
	}
	if b.MenuState == MenuTarget {
		b.drawTargetList(screen)
	}

	b.drawHitboxes(screen)
}

func (b *Battle) drawMainButtons(screen *ebiten.Image) {
	totalW := b.calcButtonsWidth()
	screenW := float64(screen.Bounds().Dx())

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

	actStartY := menuY + menuBtnH - 5
	actLineH := 60.0

	if b.SoulSprite != nil {
		soulX := 60.0
		soulY := actStartY + float64(b.SelectedAct)*actLineH + 8
		b.SoulSprite.Draw(screen, soulX, soulY, 2.0, 2.0, 0)
	}

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
			names = append(names, m.Name)
		}
	} else {
		for _, o := range b.Opponents {
			names = append(names, o.Name)
		}
	}

	if len(names) == 0 {
		return
	}

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
		b.drawMenuString(screen, targetStyle, name, listStartX+2, y-32)
	}
}
