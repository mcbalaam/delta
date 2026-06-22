package arena

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
)

type SquareArena struct {
	CenterX float64
	CenterY float64

	TopWall    *ArenaWall
	BottomWall *ArenaWall
	LeftWall   *ArenaWall
	RightWall  *ArenaWall
	Walls      []*ArenaWall

	Sprite  *render.AnimatedIcon
	SpriteX float64 // background sprite draw position (independent of arena geometry)
	SpriteY float64
	Scale   float64 // sprite render scale
}

func NewSquareArena(centerX, centerY float64) *SquareArena {
	size := 250.0
	halfSize := size / 2
	wallThickness := 6.0

	green := color.RGBA{0, 255, 0, 255}

	a := &SquareArena{
		CenterX: centerX,
		CenterY: centerY,

		TopWall: NewArenaWall(
			centerX, centerY-halfSize,
			size+6, wallThickness,
			green,
		),

		BottomWall: NewArenaWall(
			centerX, centerY+halfSize,
			size+6, wallThickness,
			green,
		),

		LeftWall: NewArenaWall(
			centerX-halfSize, centerY,
			wallThickness, size+6,
			green,
		),

		RightWall: NewArenaWall(
			centerX+halfSize, centerY,
			wallThickness, size+6,
			green,
		),
	}

	a.Walls = []*ArenaWall{
		a.TopWall,
		a.BottomWall,
		a.LeftWall,
		a.RightWall,
	}

	return a
}

func (sa *SquareArena) SetSpritePos(x, y, s float64) {
	sa.SpriteX = x
	sa.SpriteY = y
	sa.Scale = s
}

func (sa *SquareArena) SetSprite(s *render.AnimatedIcon) {
	sa.Sprite = s
}

func (sa *SquareArena) ArenaInner() (x, y, w, h float64) {
	if sa.TopWall == nil {
		return 0, 0, 0, 0
	}
	left := sa.LeftWall.Transform.X + sa.LeftWall.Collider.Width/2
	top := sa.TopWall.Transform.Y + sa.TopWall.Collider.Height/2
	right := sa.RightWall.Transform.X - sa.RightWall.Collider.Width/2
	bottom := sa.BottomWall.Transform.Y - sa.BottomWall.Collider.Height/2
	return left, top, right - left, bottom - top
}

func (sa *SquareArena) CheckCollision(obj *engine.Entity) *ArenaWall {
	for _, wall := range sa.Walls {
		if obj.Collider != nil && wall.Collider != nil {
			if obj.Collider.CollidesWith(wall.Collider) {
				return wall
			}
		}
	}
	return nil
}

func (sa *SquareArena) GetWallByName(name string) *ArenaWall {
	switch name {
	case "top":
		return sa.TopWall
	case "bottom":
		return sa.BottomWall
	case "left":
		return sa.LeftWall
	case "right":
		return sa.RightWall
	}
	return nil
}

// ── Queueable render layers ──────────────────────────────────────

// SpriteLayer renders the arena background sprite via the draw/update queues.
type SpriteLayer struct {
	Arena   *SquareArena
	Visible bool
}

func (l *SpriteLayer) Draw(screen *ebiten.Image) {
	if !l.Visible || l.Arena.Sprite == nil {
		return
	}
	sc := l.Arena.Scale
	if sc == 0 {
		sc = 2
	}
	l.Arena.Sprite.Draw(screen, l.Arena.SpriteX, l.Arena.SpriteY, sc, sc, 0)
}

func (l *SpriteLayer) Update(dt time.Duration) {
	if !l.Visible || l.Arena.Sprite == nil {
		return
	}
	l.Arena.Sprite.Update(dt)
}

// BoxLayer renders the black interior fill and green walls via the draw queue.
type BoxLayer struct {
	Arena   *SquareArena
	Visible bool
}

func (l *BoxLayer) Draw(screen *ebiten.Image) {
	if !l.Visible {
		return
	}
	ix, iy, iw, ih := l.Arena.ArenaInner()
	vector.DrawFilledRect(screen,
		float32(ix), float32(iy), float32(iw), float32(ih),
		color.Black, false)

	for _, wall := range l.Arena.Walls {
		wall.Draw(screen)
	}
}
