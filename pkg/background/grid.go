package background

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// hex parses an RGBA hex string like "#RRGGBBAA" into color.RGBA.
func hex(s string) color.RGBA {
	if len(s) < 7 || s[0] != '#' {
		return color.RGBA{}
	}
	parse := func(start, end int) uint8 {
		var v uint8
		for i := start; i < end; i++ {
			v <<= 4
			c := s[i]
			switch {
			case c >= '0' && c <= '9':
				v |= uint8(c - '0')
			case c >= 'A' && c <= 'F':
				v |= uint8(c - 'A' + 10)
			case c >= 'a' && c <= 'f':
				v |= uint8(c - 'a' + 10)
			default:
				return 0
			}
		}
		return v
	}
	r := parse(1, 3)
	g := parse(3, 5)
	b := parse(5, 7)
	a := uint8(255)
	if len(s) >= 9 {
		a = parse(7, 9)
	}
	return color.RGBA{r, g, b, a}
}

// DualGridBG renders two square grids at different speeds for a parallax
// Deltarune-style background.
type DualGridBG struct {
	offsetFast float64
	offsetSlow float64

	spacing     float64 // grid cell size in pixels
	brightColor color.Color
	darkColor   color.Color
	bgColor     color.Color
}

// GridLayer wraps DualGridBG for use with engine.DefaultQueue / DefaultUpdateQueue.
type GridLayer struct {
	BG *DualGridBG
}

func (l *GridLayer) Draw(screen *ebiten.Image) {
	l.BG.Draw(screen)
}

func (l *GridLayer) Update(dt time.Duration) {
	l.BG.Update(dt)
}

func NewDualGridBG() *DualGridBG {
	return &DualGridBG{
		spacing:     96,
		brightColor: hex("#42004230"),
		darkColor:   hex("#24002430"),
		bgColor:     hex("#000000FF"),
	}
}

func (d *DualGridBG) Update(dt time.Duration) {
	sec := dt.Seconds()
	d.offsetSlow -= 30 * sec
	d.offsetFast += 90 * sec
}

func (d *DualGridBG) Draw(screen *ebiten.Image) {
	bounds := screen.Bounds()
	w := float64(bounds.Dx())
	h := float64(bounds.Dy())

	// Fill background
	vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), d.bgColor, false)

	d.drawGrid(screen, w, h, d.offsetSlow, d.darkColor, 1)
	d.drawGrid(screen, w, h, d.offsetFast, d.brightColor, 1)
}

func (d *DualGridBG) drawGrid(screen *ebiten.Image, w, h, offset float64, clr color.Color, lineWidth float32) {
	ox := int(offset) % int(d.spacing)
	oy := int(offset*0.7) % int(d.spacing)

	for x := -ox; x < int(w); x += int(d.spacing) {
		xf := float32(x)
		vector.StrokeLine(screen, xf, 0, xf, float32(h), lineWidth, clr, false)
	}

	for y := -oy; y < int(h); y += int(d.spacing) {
		yf := float32(y)
		vector.StrokeLine(screen, 0, yf, float32(w), yf, lineWidth, clr, false)
	}
}
