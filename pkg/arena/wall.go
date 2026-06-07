package arena

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/mcbalaam/delta/internal/engine"
)

type ArenaWall struct {
	*engine.RigidObject
	Color color.Color
}

func NewArenaWall(x, y, width, height float64, color color.Color) *ArenaWall {
	obj := &ArenaWall{
		RigidObject: engine.NewRigidObject(
			x, y,
			0, 0,
			1, 1,
			0,
			nil,
			width, height,
			0, 0,
		),
		Color: color,
	}
	return obj
}

func (w *ArenaWall) Draw(s *ebiten.Image) {
	if w.Hitbox == nil {
		return
	}

	vector.DrawFilledRect(s,
		float32(w.PosX-w.Hitbox.Width/2),
		float32(w.PosY-w.Hitbox.Height/2),
		float32(w.Hitbox.Width),
		float32(w.Hitbox.Height),
		w.Color, false)
}
