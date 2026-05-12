package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/mcbalaam/delta/internal/render"
)

var icon *render.AnimatedIcon

type Game struct {
	last time.Time
}

var xPos, yPos float64 = 200, 200

func init() {
	var err error
	icon, err = render.NewAnimatedIconFromPath("media/sprites/baba", "baba_right")
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func (g *Game) Update() error {
	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		icon.SetIconState("baba_right")
		xPos += 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		icon.SetIconState("baba_left")
		xPos -= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		yPos -= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		yPos += 2
	}

	if icon != nil {
		icon.Update(dt)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if icon != nil {
		icon.Draw(screen, xPos, yPos, 2, 2, 0)
	}
	ebitenutil.DebugPrint(screen, icon.CurrentState.Name)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Animated Icon")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
