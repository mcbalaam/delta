package battle

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/components"
)

// ── Soul movement ────────────────────────────────────────────────

func (b *Battle) updateSoulMovement(dt time.Duration) {
	if b.soulFlyoutPlaying {
		b.soulFlyoutProgress += dt.Seconds() / b.soulFlyoutDuration
		if b.soulFlyoutProgress >= 1 {
			b.soulFlyoutProgress = 1
			b.soulFlyoutPlaying = false
		}
		t := b.soulFlyoutProgress
		t = t * (2 - t)
		b.SoulX = b.soulFlyoutStartX + (b.soulFlyoutTargetX-b.soulFlyoutStartX)*t
		b.SoulY = b.soulFlyoutStartY + (b.soulFlyoutTargetY-b.soulFlyoutStartY)*t
		if b.SoulCollider != nil {
			ct := &components.Transform{X: b.SoulX, Y: b.SoulY, ScaleX: 2, ScaleY: 2}
			b.SoulCollider.UpdateWorldVerts(ct)
		}
		return
	}

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

func (b *Battle) enterBoxOpen() {
	if b.showArena != nil {
		b.showArena()
	}
	b.boxOpenTimer = 0
	if b.boxOpenDuration == 0 {
		b.boxOpenDuration = 0.5
	}

	if b.ArenaBoundsW > 0 && b.ArenaBoundsH > 0 {
		targetX := b.ArenaBoundsX + b.ArenaBoundsW/2
		targetY := b.ArenaBoundsY + b.ArenaBoundsH/2

		leaderIdx := -1
		for i, m := range b.Party {
			if m.IsLeader && m.Alive() {
				leaderIdx = i
				break
			}
		}

		if leaderIdx >= 0 {
			lx, ly := b.partyMemberScreenPos(leaderIdx)
			b.SoulX = lx
			b.SoulY = ly
			b.soulFlyoutStartX = lx
			b.soulFlyoutStartY = ly
			b.soulFlyoutTargetX = targetX
			b.soulFlyoutTargetY = targetY
			b.soulFlyoutPlaying = true
			b.soulFlyoutProgress = 0
			b.soulFlyoutDuration = 0.5
		} else {
			b.SoulX = targetX
			b.SoulY = targetY
		}

		if b.SoulCollider != nil {
			t := &components.Transform{X: b.SoulX, Y: b.SoulY, ScaleX: 2, ScaleY: 2}
			b.SoulCollider.UpdateWorldVerts(t)
		}
	}
}

func (b *Battle) enterBoxClose() {
	b.boxCloseTimer = 0
	if b.boxCloseDuration == 0 {
		b.boxCloseDuration = 0.35
	}
	b.turnAttackSeq = nil

	if b.startExitArena != nil {
		b.startExitArena()
	}

	leaderIdx := -1
	for i, m := range b.Party {
		if m.IsLeader && m.Alive() {
			leaderIdx = i
			break
		}
	}
	if leaderIdx >= 0 {
		tx, ty := b.partyMemberScreenPos(leaderIdx)
		b.soulFlybackStartX = b.SoulX
		b.soulFlybackStartY = b.SoulY
		b.soulFlybackTargetX = tx
		b.soulFlybackTargetY = ty
	} else {
		b.soulFlybackStartX = b.SoulX
		b.soulFlybackStartY = b.SoulY
		b.soulFlybackTargetX = b.SoulX
		b.soulFlybackTargetY = b.SoulY
	}
}

func (b *Battle) updateBoxOpen(dt time.Duration) {
	b.boxOpenTimer += dt.Seconds()
	if b.boxOpenTimer >= b.boxOpenDuration {
		b.SetState(StateEnemyTurn)
	}
}

func (b *Battle) updateBoxClose(dt time.Duration) {
	b.boxCloseTimer += dt.Seconds()
	t := b.boxCloseTimer / b.boxCloseDuration
	if t > 1 {
		t = 1
	}
	et := t * t
	b.SoulX = b.soulFlybackStartX + (b.soulFlybackTargetX-b.soulFlybackStartX)*et
	b.SoulY = b.soulFlybackStartY + (b.soulFlybackTargetY-b.soulFlybackStartY)*et
	if b.SoulCollider != nil {
		ct := &components.Transform{X: b.SoulX, Y: b.SoulY, ScaleX: 2, ScaleY: 2}
		b.SoulCollider.UpdateWorldVerts(ct)
	}

	if t >= 1 {
		b.SoulX = b.soulFlybackTargetX
		b.SoulY = b.soulFlybackTargetY
		if b.hideArena != nil {
			b.hideArena()
		}
		for i := 0; i < len(b.Party); i++ {
			if b.Party[i].Alive() {
				b.SetState(selectingStateFor(i))
				return
			}
		}
		b.SetState(StateEnemyTarget)
	}
}
