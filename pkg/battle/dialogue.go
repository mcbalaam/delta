package battle

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/queues"
)

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
