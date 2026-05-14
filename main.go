package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/game"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/systems"
)

type Game struct {
	last time.Time
}

var baba *engine.RigidObject

func init() {
	var err error
	icon, err := render.NewAnimatedIconFromPath("media/sprites/baba", "baba_left")
	baba = engine.NewRigidObject(200, 200, 0, 0, 2, 2, 0, *icon, 24, 24, 0, 0)
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

	var projectile *game.Projectile

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		icon, err := render.NewAnimatedIconFromPath("media/sprites/attack", "attack_idle")
		if err != nil {
			log.Fatalf("%v", err)
		}
		projectile = game.NewProjectile(400, 400, 180, 180, 2, 2, 45, *icon, 10, 10, -10, -18, 20, true, time.Duration(2222))
		systems.MasterSignalBus.Subscribe("collision", projectile, func(signal systems.Signal) {
			println("collision detected")
		})
	}

	engine.DefaultUpdateQueue.Execute(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	engine.DefaultQueue.Execute(screen)
	ebitenutil.DebugPrint(screen, baba.Icon.CurrentState.Name)
	ebitenutil.DebugPrint(screen, "\nhitboxes shown")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 960
}

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Animated Icon")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
