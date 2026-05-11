package main

import (
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var img *ebiten.Image

func init() {
	var err error
	img, _, err = ebitenutil.NewImageFromFile("gopher.png")
	if err != nil {
		log.Fatal(err)
	}
}

type Game struct {
	State string
}

var x float64 = 200
var y float64 = 200

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.Key(ebiten.KeyRight)) {
		x += 2
	}
	if ebiten.IsKeyPressed(ebiten.Key(ebiten.KeyLeft)) {
		x -= 2
	}
	if ebiten.IsKeyPressed(ebiten.Key(ebiten.KeyUp)) {
		y -= 2
	}
	if ebiten.IsKeyPressed(ebiten.Key(ebiten.KeyDown)) {
		y += 2
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Render an image")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
