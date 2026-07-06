package battle

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mcbalaam/delta/internal/engine/queues"
)

// ── Turn wait / attack update / target selection ─────────────────

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

	b.turnAttackSeq.Update(dt, b.SoulX, b.SoulY)
	b.turnAttackElapsed += dt

	b.InvincibilityTimer -= dt.Seconds()
	if b.InvincibilityTimer < 0 {
		b.InvincibilityTimer = 0
	}

	if b.SoulCollider != nil && len(b.Targets) > 0 {
		alive := b.turnAttackSeq.ActiveProjectiles[:0]
		for _, p := range b.turnAttackSeq.ActiveProjectiles {
			if p.Collider != nil && p.Transform != nil && b.SoulCollider.CollidesWith(p.Collider) {
				if b.InvincibilityTimer <= 0 {
					for _, tIdx := range b.Targets {
						if tIdx >= 0 && tIdx < len(b.Party) && b.Party[tIdx].Alive() {
							b.Party[tIdx].TakeDamage(b, p.Damage)
							break
						}
					}
					b.InvincibilityTimer = b.InvincibilityDuration
				}
				if p.Fragile {
					queues.QDel(p)
					continue
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
			for _, ot := range b.Targets {
				if ot == j {
					already = true
					break
				}
			}
			if !already {
				b.Targets[i] = j
				return
			}
		}
		b.Targets = append(b.Targets[:i], b.Targets[i+1:]...)
		return
	}
}
