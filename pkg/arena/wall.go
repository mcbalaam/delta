package arena

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/components"
	"github.com/mcbalaam/delta/internal/engine/queues"
)

type ArenaWall struct {
	engine.Entity
	Color color.Color
}

func NewArenaWall(x, y, width, height float64, color color.Color) *ArenaWall {
	wall := &ArenaWall{Color: color}
	wall.Layer = queues.LayerArena
	wall.Transform = &components.Transform{X: x, Y: y, ScaleX: 1, ScaleY: 1}
	wall.Collider = components.NewCollider(width, height, 0, 0)
	wall.Collider.UpdateWorldVerts(wall.Transform)
	queues.DefaultUpdateQueue.Schedule(wall)
	return wall
}

func (w *ArenaWall) Draw(s *ebiten.Image) {
	if w.Collider == nil {
		return
	}

	vector.DrawFilledRect(s,
		float32(w.Transform.X-w.Collider.Width/2),
		float32(w.Transform.Y-w.Collider.Height/2),
		float32(w.Collider.Width),
		float32(w.Collider.Height),
		w.Color, false)
}
