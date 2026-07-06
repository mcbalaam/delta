package background

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ── Snowflake constants ──────────────────────────────────────────

const (
	snowSpacing  = 120.0       // grid cell size
	snowRadius   = 36.0        // arm length (main stick)
	snowSpeed    = 30.0        // px/s movement speed along x
	snowRotSpeed = 3.0         // rotation frequency (rad/s)
	snowRotAmpl  = math.Pi / 6 // ±30° rotation amplitude
	snowLineW    = 1.0         // line thickness
)

// SnowBG draws a field of rotating 8-point snowflake stars moving
// diagonally (y = 2·x) and rotating with ease-in/ease-out.
type SnowBG struct {
	elapsed time.Duration
}

type SnowLayer struct {
	BG *SnowBG
}

func (l *SnowLayer) Draw(screen *ebiten.Image) { l.BG.Draw(screen) }
func (l *SnowLayer) Update(dt time.Duration)   { l.BG.Update(dt) }

func NewSnowBG() *SnowBG {
	return &SnowBG{}
}

func (s *SnowBG) Update(dt time.Duration) {
	s.elapsed += dt
}

func (s *SnowBG) Draw(screen *ebiten.Image) {
	bounds := screen.Bounds()
	w, h := float64(bounds.Dx()), float64(bounds.Dy())
	t := s.elapsed.Seconds()

	// Black background
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), color.Black, false)

	// Rotation angle — sine gives ease-in/ease-out at direction changes
	rot := snowRotAmpl * math.Sin(t*snowRotSpeed)

	// Offset so the field scrolls along y = 2x
	ox := math.Mod(t*snowSpeed, snowSpacing)
	oy := math.Mod(t*snowSpeed*2, snowSpacing) // y = 2x

	clr := color.RGBA{R: 60, G: 140, B: 215, A: 10}

	// Draw a snowflake at each grid intersection
	for x := -ox; x < w+snowSpacing; x += snowSpacing {
		for y := -oy; y < h+snowSpacing; y += snowSpacing {
			drawSnowflake(screen, x, y, snowRadius, rot, clr)
		}
	}
}

// drawSnowflake draws an 8-pointed star at (cx, cy) rotated by angle,
// with V-branches at the tip of each arm for a fluffy look.
func drawSnowflake(screen *ebiten.Image, cx, cy, r, angle float64, clr color.Color) {
	sinA, cosA := math.Sincos(angle)
	lineW := float32(snowLineW)

	// 8 arm directions (normalised)
	arms := [8][2]float64{
		{0, -1}, {1, -1}, {1, 0}, {1, 1},
		{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	}
	const invSqrt2 = 0.7071067811865475
	for i := range arms {
		if arms[i][0] != 0 && arms[i][1] != 0 {
			arms[i][0] *= invSqrt2
			arms[i][1] *= invSqrt2
		}
	}

	// Branch angle offset from main arm (±30°)
	const branchAngle = math.Pi / 6 // 30°
	branchLen := r * 0.45

	for _, a := range arms {
		// Main arm (2× radius, half-length for the core part)
		dx := a[0]*r*cosA - a[1]*r*sinA
		dy := a[0]*r*sinA + a[1]*r*cosA
		tipX, tipY := cx+dx, cy+dy

		vector.StrokeLine(screen,
			float32(cx), float32(cy),
			float32(tipX), float32(tipY),
			lineW, clr, false,
		)

		// V-branch at tip: two short lines at ±30° from the arm direction
		adx := dx / r // arm unit direction
		ady := dy / r
		for _, sign := range []float64{1, -1} {
			bSin, bCos := math.Sincos(sign * branchAngle)
			bdx := (adx*bCos - ady*bSin) * branchLen
			bdy := (adx*bSin + ady*bCos) * branchLen
			vector.StrokeLine(screen,
				float32(tipX), float32(tipY),
				float32(tipX+bdx), float32(tipY+bdy),
				lineW, clr, false,
			)
		}
	}
}
