package arena

import (
	"image/color"
	"math"
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

type GhostFrame struct {
	Scale    float64
	Angle    float64
	Recorded float64
}

const maxGhosts = 12
const boxImgSize = 280

// BoxLayer renders the black interior fill and green walls via the draw queue.
type BoxLayer struct {
	Arena   *SquareArena
	Visible bool

	entrancePlaying  bool
	entranceProgress float64
	entranceDuration float64
	ghostFrames      [maxGhosts]GhostFrame
	ghostHead        int
	ghostCount       int
	recordSkip       int

	exitPlaying  bool
	exitProgress float64
	exitDuration float64

	boxImg *ebiten.Image
}

func (l *BoxLayer) ensureBoxImg() {
	if l.boxImg != nil {
		return
	}
	l.boxImg = ebiten.NewImage(boxImgSize, boxImgSize)
	_, _, iw, ih := l.Arena.ArenaInner()
	imgCx, imgCy := float64(boxImgSize)/2, float64(boxImgSize)/2

	innerX := imgCx - iw/2
	innerY := imgCy - ih/2

	l.boxImg.Fill(color.Transparent)

	vector.DrawFilledRect(l.boxImg, float32(innerX), float32(innerY), float32(iw), float32(ih),
		color.Black, false)

	wt := 6.0
	green := color.RGBA{0, 255, 0, 255}
	vector.DrawFilledRect(l.boxImg, float32(innerX-wt), float32(innerY-wt), float32(iw+2*wt), float32(wt), green, false)
	vector.DrawFilledRect(l.boxImg, float32(innerX-wt), float32(innerY+ih), float32(iw+2*wt), float32(wt), green, false)
	vector.DrawFilledRect(l.boxImg, float32(innerX-wt), float32(innerY), float32(wt), float32(ih), green, false)
	vector.DrawFilledRect(l.boxImg, float32(innerX+iw), float32(innerY), float32(wt), float32(ih), green, false)
}

func (l *BoxLayer) StartEntrance() {
	l.entrancePlaying = true
	l.entranceProgress = 0
	l.ghostHead = 0
	l.ghostCount = 0
	l.recordSkip = 0
	if l.entranceDuration == 0 {
		l.entranceDuration = 0.4
	}
}

func entranceScale(t float64) float64 {
	if t >= 1 {
		return 1
	}
	const c1 = 0.30158
	const c3 = c1 + 1
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func entranceAngle(t float64) float64 {
	if t >= 1 {
		return 0
	}
	dt := 1 - t
	return dt * dt * 2 * math.Pi
}

func exitScale(t float64) float64 {
	if t >= 1 {
		return 0
	}
	return math.Pow(1-t, 3)
}

func exitAngle(t float64) float64 {
	if t >= 1 {
		return 0
	}
	return t * t * 4 * math.Pi
}

func (l *BoxLayer) StartExit() {
	l.exitPlaying = true
	l.exitProgress = 0
	if l.exitDuration == 0 {
		l.exitDuration = 0.35
	}
}

func (l *BoxLayer) Update(dt time.Duration) {
	if l.exitPlaying {
		l.exitProgress += dt.Seconds() / l.exitDuration
		if l.exitProgress >= 1 {
			l.exitProgress = 1
			l.exitPlaying = false
			l.Visible = false
		}
		return
	}

	if !l.Visible || !l.entrancePlaying {
		return
	}

	l.recordSkip++
	if l.recordSkip%2 == 0 {
		l.ghostFrames[l.ghostHead] = GhostFrame{
			Scale:    entranceScale(l.entranceProgress),
			Angle:    entranceAngle(l.entranceProgress),
			Recorded: l.entranceProgress,
		}
		l.ghostHead = (l.ghostHead + 1) % maxGhosts
		if l.ghostCount < maxGhosts {
			l.ghostCount++
		}
	}

	l.entranceProgress += dt.Seconds() / l.entranceDuration
	if l.entranceProgress >= 1 {
		l.entranceProgress = 1
		l.entrancePlaying = false
	}
}

func (l *BoxLayer) drawTransformedBox(screen *ebiten.Image, scale, angle, alpha float64) {
	l.ensureBoxImg()

	ix, iy, iw, ih := l.Arena.ArenaInner()
	cx, cy := ix+iw/2, iy+ih/2
	imgCx := float64(boxImgSize) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-imgCx, -imgCx)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(cx, cy)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(l.boxImg, op)
}

func (l *BoxLayer) Draw(screen *ebiten.Image) {
	if l.exitPlaying {
		scale := exitScale(l.exitProgress)
		angle := exitAngle(l.exitProgress)
		l.drawTransformedBox(screen, scale, angle, 1.0)
		return
	}

	if !l.Visible {
		return
	}

	if l.entrancePlaying || l.entranceProgress < 1 {
		currScale := entranceScale(l.entranceProgress)
		currAngle := entranceAngle(l.entranceProgress)

		l.drawTransformedBox(screen, currScale, currAngle, 1.0)

		count := l.ghostCount
		if count > 0 {
			start := (l.ghostHead - count + maxGhosts) % maxGhosts
			for i := 0; i < count; i++ {
				idx := (start + i) % maxGhosts
				g := l.ghostFrames[idx]
				age := l.entranceProgress - g.Recorded
				if age < 0 {
					age = 0
				}
				ghostAlpha := (1.0 - age) * 0.35
				if ghostAlpha > 0 {
					l.drawTransformedBox(screen, g.Scale, g.Angle, ghostAlpha)
				}
			}
		}

		return
	}

	ix, iy, iw, ih := l.Arena.ArenaInner()
	vector.DrawFilledRect(screen, float32(ix), float32(iy), float32(iw), float32(ih),
		color.Black, false)
	for _, wall := range l.Arena.Walls {
		wall.Draw(screen)
	}
}
