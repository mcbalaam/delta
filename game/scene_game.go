package game

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/pkg/battle"
)

type GameScene struct {
	Battle          *battle.Battle
	dialogueBoxID   interface{}
}

func (s *GameScene) Update(dt time.Duration) {
	if s.Battle != nil {
		s.Battle.Update(dt)
		s.Battle.NavigateMenu()
	}
	queues.DefaultUpdateQueue.Execute(dt)
	queues.DefaultDeleteQueue.Execute()

	// Schedule/un-schedule dialogue box at layer between arena and text
	if s.Battle != nil {
		s.Battle.ScheduleDialogueBox(&s.dialogueBoxID)
	}
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	queues.DefaultQueue.Execute(screen)
	if s.Battle != nil {
		s.Battle.DrawMenu(screen)
	}
}
