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

	Sprite *render.AnimatedIcon
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

func (sa *SquareArena) SetSprite(s *render.AnimatedIcon) {
	sa.Sprite = s
}

func (sa *SquareArena) ArenaInner() (x, y, w, h float64) {
	if sa.TopWall == nil {
		return 0, 0, 0, 0
	}
	left := sa.LeftWall.PosX + sa.LeftWall.Hitbox.Width/2
	top := sa.TopWall.PosY + sa.TopWall.Hitbox.Height/2
	right := sa.RightWall.PosX - sa.RightWall.Hitbox.Width/2
	bottom := sa.BottomWall.PosY - sa.BottomWall.Hitbox.Height/2
	return left, top, right - left, bottom - top
}

func (sa *SquareArena) CheckCollision(obj *engine.RigidObject) *ArenaWall {
	for _, wall := range sa.Walls {
		if obj.CollidesWith(wall.RigidObject) {
			return wall
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
	l.Arena.Sprite.Draw(screen, l.Arena.CenterX-640, l.Arena.CenterY-480, 2, 2, 0)
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
