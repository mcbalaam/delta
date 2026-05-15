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
	if w.Hitbox == nil || len(w.Hitbox.WorldVerts) < 2 {
		return
	}

	for i := 0; i < len(w.Hitbox.WorldVerts); i++ {
		current := w.Hitbox.WorldVerts[i]
		next := w.Hitbox.WorldVerts[(i+1)%len(w.Hitbox.WorldVerts)]

		vector.StrokeLine(s,
			float32(current.X), float32(current.Y),
			float32(next.X), float32(next.Y),
			3,
			w.Color,
			true)
	}

	w.DrawHitboxDebug(s)
}
