package battle

import (
	"time"

	"github.com/mcbalaam/delta/internal/render"
)

// ── Character animations ─────────────────────────────────────────

func (b *Battle) updateCharacterAnimations(dt time.Duration) {
	for _, m := range b.Party {
		if m.CharacterSprite == nil {
			continue
		}
		cs := m.CharacterSprite
		if m.IsDefending {
			if cs.CurrentState.Name != "defend" {
				cs.SetIconState("defend")
				cs.CurrentState.Mode = render.AnimationModeOnce
			}
			cs.Update(dt)
		} else {
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
