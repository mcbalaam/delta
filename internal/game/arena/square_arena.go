package arena

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine"
)

type SquareArena struct {
	TopWall    *ArenaWall
	BottomWall *ArenaWall
	LeftWall   *ArenaWall
	RightWall  *ArenaWall
	Walls      []*ArenaWall
}

func NewSquareArena(centerX, centerY float64) *SquareArena {
	size := 200.0
	halfSize := size / 2
	wallThickness := 10.0

	green := color.RGBA{0, 255, 0, 255}

	arena := &SquareArena{
		TopWall: NewArenaWall(
			centerX, centerY-halfSize,
			size, wallThickness,
			green,
		),

		BottomWall: NewArenaWall(
			centerX, centerY+halfSize,
			size, wallThickness,
			green,
		),

		LeftWall: NewArenaWall(
			centerX-halfSize, centerY,
			wallThickness, size,
			green,
		),

		RightWall: NewArenaWall(
			centerX+halfSize, centerY,
			wallThickness, size,
			green,
		),
	}

	arena.Walls = []*ArenaWall{
		arena.TopWall,
		arena.BottomWall,
		arena.LeftWall,
		arena.RightWall,
	}

	return arena
}

func (sa *SquareArena) Update(dt float64) {
	for _, wall := range sa.Walls {
		_ = wall
	}
}

func (sa *SquareArena) Draw(screen *ebiten.Image) {
	for _, wall := range sa.Walls {
		wall.Draw(screen)
	}
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
