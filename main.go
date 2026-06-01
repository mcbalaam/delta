package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/mcbalaam/delta/internal/assets"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/game/arena"
	"github.com/mcbalaam/delta/internal/game/battle"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
	"github.com/mcbalaam/delta/internal/systems"
)

type Game struct {
	last              time.Time
	arena             *arena.SquareArena
	collisionListener bool
	soundPlayer       *sound.SoundPlayer
	textEngine        *engine.TextEngine
	CurrentBattle     *battle.Battle
}

var soul *engine.RigidObject

func init() {
	var err error
	icon, err := render.NewAnimatedIconFromPath("media/sprites/soul", "idle")
	if err != nil {
		log.Fatalf("%v", err)
	}
	soul = engine.NewRigidObject(640, 480, 0, 0, 2, 2, 0, icon, 12, 12, -4, -4)
	assets.ProcessFonts()
}

func (g *Game) Update() error {
	if !g.collisionListener {
		systems.MasterSignalBus.Subscribe("collision:arena-wall", soul, func(signal systems.Signal) {
			println("COLLISION DETECTED")
		})
		g.collisionListener = true
	}

	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now

	oldX := soul.PosX
	oldY := soul.PosY

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		soul.PosX += 3
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		soul.PosX -= 3
	}

	soul.UpdateHitbox()
	if g.arena.CheckCollision(soul) != nil {
		soul.PosX = oldX
		soul.UpdateHitbox()
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		soul.PosY -= 3
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		soul.PosY += 3
	}

	soul.UpdateHitbox()
	if g.arena.CheckCollision(soul) != nil {
		soul.PosY = oldY
		soul.UpdateHitbox()
	}
	engine.DefaultUpdateQueue.Execute(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.arena.Draw(screen)

	engine.DefaultQueue.Execute(screen)
	ebitenutil.DebugPrint(screen, "hitboxes shown")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 960
}

func main() {
	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("Animated Icon")

	soundPlayer, err := sound.NewSoundPlayer(44000)
	if err != nil {
		log.Fatalf("Failed to initialize sound player: %v", err)
	}
	defer soundPlayer.Shutdown()

	soundPlayer.RegisterNewSound("media/sound/snd_text.wav", "snd_text")
	soundPlayer.RegisterNewSound("media/sound/snd_text2.wav", "snd_text2")

	textEngine := &engine.TextEngine{
		FontsLoaded: make(map[string]render.AnimatedIcon),
	}

	game := &Game{
		arena:       arena.NewSquareArena(640, 480),
		soundPlayer: soundPlayer,
		textEngine:  textEngine,
	}

	lines := []string{
		"* Ralsei chanted something!$p200$n* JEVIL's next attack weakened!$p200$n* JEVIL became more TIRED!$f",
	}
	session := engine.NewDialogueSession(textEngine, lines, engine.StyleNarrative, soundPlayer)
	session.Start()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
