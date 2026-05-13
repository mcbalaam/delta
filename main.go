package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
)

type Game struct {
	last time.Time
}

var baba *engine.RigidObject

func init() {
	var err error
	icon, err := render.NewAnimatedIconFromPath("media/sprites/baba", "baba_left")
	baba = engine.NewRigidObject(200, 200, 12, 12, 6, 6, *icon)
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
		baba.Icon.SetIconState("baba_right")
		baba.PosX += 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		baba.Icon.SetIconState("baba_left")
		baba.PosX -= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		baba.PosY -= 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		baba.PosY += 2
	}

	baba.Update(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	baba.Draw(screen)
	baba.DrawHitboxDebug(screen)
	ebitenutil.DebugPrint(screen, baba.Icon.CurrentState.Name)
	ebitenutil.DebugPrint(screen, "\nhitboxes shown")
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
