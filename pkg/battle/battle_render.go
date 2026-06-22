package battle

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mcbalaam/delta/internal/engine"
)

func (b *Battle) partyMemberScreenPos(memberIdx int) (x, y float64) {
	n := len(b.Party)
	if n == 0 {
		return 0, 0
	}

	slotOf := func(memberIdx int) int {
		switch n {
		case 1:
			return 2
		case 2:
			return memberIdx*2 + 1
		default:
			return memberIdx * 2
		}
	}

	s := slotOf(memberIdx)
	x = float64(s)*1280.0/6.0 + 1280.0/12.0 + 100
	y = 160
	return
}

func (b *Battle) drawPartyOnArena(screen *ebiten.Image) {
	for i, m := range b.Party {
		if m.CharacterSprite == nil {
			continue
		}
		x, y := b.partyMemberScreenPos(i)
		m.CharacterSprite.Draw(screen, x, y, 3, 3, 0)
	}
}

func (b *Battle) drawOpponents(screen *ebiten.Image) {
	if len(b.Opponents) == 0 {
		return
	}
	for i, o := range b.Opponents {
		if o.CharacterSprite == nil {
			continue
		}
		// Flicker glow when focused in target selection
		bright := 1.0
		if b.MenuState == MenuTarget && !b.targetIsAlly && i == b.SelectedTarget {
			bright = 1.0 + 0.5*math.Sin(b.flickerAccum*8)
			if bright < 1.0 {
				bright = 1.0
			}
		}
		o.CharacterSprite.DrawWithColorScale(screen, 950, 160, 3, 3, 0, bright, bright, bright, 1)
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
	pillarH := 70.0
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
		CharSpacing: -10,
	}

	hpStyle := engine.TextStyle{
		FontName: "vaticanius", ScaleX: 0.3, ScaleY: 0.3,
		FontHeight: 24.0, LineSpacing: 0, DefaultDelay: 0.03,
		CharSpacing: -4,
	}

	for i, m := range b.Party {
		isFocused := i == b.ActiveMember && b.State.IsSelecting()
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

		if b.HPIcon != nil {
			b.HPIcon.Draw(screen, cx+215, cardY+cardH-28, 2, 2, 0)
		}

		ebitenutil.DrawRect(screen, hpBarX, hpBarY, hpBarW, hpBarH,
			color.RGBA{40, 40, 40, 255})
		if m.MaxHP > 0 {
			fill := hpBarW * (m.HP / m.MaxHP)
			if fill < 1 && m.HP > 0 {
				fill = 1
			}
			hpClr := accent
			if m.HP < m.MaxHP*0.25 && isFocused {
				hpClr = color.RGBA{255, 60, 40, 255}
			}
			ebitenutil.DrawRect(screen, hpBarX, hpBarY, fill, hpBarH, hpClr)
		}

		b.drawMenuString(screen, nameStyle, strings.ToUpper(m.Name), cx+99, cardY-48)

		hpCur := fmt.Sprintf("%d", int(m.HP))
		hpMax := fmt.Sprintf("%d", int(m.MaxHP))
		charW := (20.0 + hpStyle.CharSpacing) * 3 * hpStyle.ScaleX
		hpCurW := float64(len(hpCur)) * charW
		hpMaxW := float64(len(hpMax)) * charW
		slashW := 12.0 * 2

		centerX := hpBarX + hpBarW/2
		slashX := centerX - slashW/2
		rightEdgeX := hpBarX + hpBarW

		b.drawMenuString(screen, hpStyle, hpCur, slashX-6-hpCurW, cardY-10)
		if b.SlashIcon != nil {
			b.SlashIcon.Draw(screen, slashX, cardY+12, 2, 2, 0)
		}
		b.drawMenuString(screen, hpStyle, hpMax, rightEdgeX-hpMaxW, cardY-10)
		if m.BattleMiniature != nil {
			m.BattleMiniature.Draw(screen, cx, cardY-5, 2, 2, 0)
		}
	}

	now := time.Now()
	const spawnInterval = 500 * time.Millisecond
	if len(b.pillarParticles) > 0 {
		idx := b.pillarParticles[0].CardIndex
		if idx != b.ActiveMember || !b.State.IsSelecting() {
			b.pillarParticles = b.pillarParticles[:0]
		}
	}
	if b.State.IsSelecting() {
		for i := range b.pillarParticles {
			p := &b.pillarParticles[i]
			if p.CardIndex < len(b.cardAnimY) {
				p.Y = b.cardAnimY[p.CardIndex] + 70 + 4 + 70
			}
		}
		b.drawPillarParticles(screen, now)

		if now.Sub(b.lastParticleSpawn) >= spawnInterval {
			b.lastParticleSpawn = now

			for i := range b.Party {
				if i != b.ActiveMember || !b.State.IsSelecting() {
					continue
				}
				cardY := 650.0
				if b.State.IsSelecting() {
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

		cutoff := now.Add(-1 * time.Second)
		aliveParticles := b.pillarParticles[:0]
		for _, p := range b.pillarParticles {
			if p.SpawnAt.After(cutoff) {
				aliveParticles = append(aliveParticles, p)
			}
		}
		b.pillarParticles = aliveParticles
	}
}

func (b *Battle) drawPillarParticles(screen *ebiten.Image, now time.Time) {
	barW := 4.0
	barH := 70.0

	for _, p := range b.pillarParticles {
		age := now.Sub(p.SpawnAt).Seconds()
		if age > 1.0 {
			continue
		}
		x := p.X + p.Dir*30*age
		y := p.Y - barH
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

func (b *Battle) drawTargetIcons(screen *ebiten.Image) {
	if b.State != StateEnemyTarget {
		return
	}
	if b.TargetIcon == nil || len(b.Targets) == 0 {
		return
	}

	for _, tIdx := range b.Targets {
		if tIdx < 0 || tIdx >= len(b.Party) {
			continue
		}
		m := b.Party[tIdx]
		if !m.Alive() || m.CharacterSprite == nil {
			continue
		}
		cx, cy := b.partyMemberScreenPos(tIdx)
		b.TargetIcon.Draw(screen, cx-10, cy+45, 3, 3, 0)
	}
}

func (b *Battle) drawDialogueBox(screen *ebiten.Image) {
	if b.State != StateEnemyTarget {
		return
	}
	if b.DialogueBoxIcon == nil {
		return
	}

	b.DialogueBoxIcon.Draw(screen, 670, 210, 2, 2, 0)
}

func (b *Battle) drawHitboxes(screen *ebiten.Image) {
	if !b.ShowHitboxes {
		return
	}

	// Soul
	if b.SoulCollider != nil && b.State == StateEnemyTurn {
		b.SoulCollider.DrawDebug(screen, color.RGBA{255, 0, 0, 255})
	}

	// Active projectiles
	if b.turnAttackSeq != nil {
		for _, p := range b.turnAttackSeq.ActiveProjectiles {
			if p.Collider != nil {
				p.Collider.DrawDebug(screen, color.RGBA{0, 255, 0, 255})
			}
		}
	}

	// Arena bounds
	if b.ArenaBoundsW > 0 {
		ebitenutil.DrawRect(screen, b.ArenaBoundsX, b.ArenaBoundsY, b.ArenaBoundsW, b.ArenaBoundsH, color.RGBA{0, 0, 255, 64})
	}
}

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
