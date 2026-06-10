package game

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
	"github.com/mcbalaam/delta/pkg/arena"
	"github.com/mcbalaam/delta/pkg/background"
	"github.com/mcbalaam/delta/pkg/battle"
)

type Game struct {
	last          time.Time
	scenes        *engine.SceneManager
	soundPlayer   *sound.SoundPlayer
	textEngine    *engine.TextEngine
	layoutOverlay *ebiten.Image
}

func NewGame(soundPlayer *sound.SoundPlayer, textEngine *engine.TextEngine) *Game {
	g := &Game{
		scenes:      &engine.SceneManager{},
		soundPlayer: soundPlayer,
		textEngine:  textEngine,
	}
	g.setupTestBattle()
	return g
}

func (g *Game) setupTestBattle() {
	menuIcon, err := render.NewAnimatedIconFromPath("media/sprites/menu", "fight")
	if err != nil {
		log.Fatalf("menu sprite: %v", err)
	}

	soulIcon, err := render.NewAnimatedIconFromPath("media/sprites/soul", "idle")
	if err != nil {
		log.Fatalf("soul sprite: %v", err)
	}

	arenaIcon, err := render.NewAnimatedIconFromPath("media/sprites/arena", "idle")
	if err != nil {
		log.Fatalf("arena sprite: %v", err)
	}

	layoutImg, _, err := ebitenutil.NewImageFromFile("media/sprites/layout.png")
	if err != nil {
		log.Printf("layout.png not found, skipping overlay: %v", err)
	}
	g.layoutOverlay = layoutImg

	krisIcon, err := render.NewAnimatedIconFromPath("game/media/sprites/kris_interface", "idle")
	if err != nil {
		log.Fatalf("kris sprite: %v", err)
	}

	susieIcon, err := render.NewAnimatedIconFromPath("game/media/sprites/susie_interface", "idle")
	if err != nil {
		log.Fatalf("kris sprite: %v", err)
	}

	ralseiIcon, err := render.NewAnimatedIconFromPath("game/media/sprites/ralsei_interface", "idle")
	if err != nil {
		log.Fatalf("kris sprite: %v", err)
	}

	bgGrid := background.NewDualGridBG()
	queues.DefaultQueue.ScheduleAt(&background.GridLayer{BG: bgGrid}, queues.LayerBackground)
	queues.DefaultUpdateQueue.Schedule(&background.GridLayer{BG: bgGrid})

	sa := arena.NewSquareArena(640, 480)
	sa.SetSprite(arenaIcon)

	spriteLayer := &arena.SpriteLayer{Arena: sa, Visible: true}
	boxLayer := &arena.BoxLayer{Arena: sa, Visible: false}
	queues.DefaultQueue.ScheduleAt(spriteLayer, queues.LayerArena)
	queues.DefaultQueue.ScheduleAt(boxLayer, queues.LayerArena)
	queues.DefaultUpdateQueue.Schedule(spriteLayer)

	showArena := func() {
		boxLayer.Visible = true
	}
	hideArena := func() {
		boxLayer.Visible = false
	}

	kris := &battle.PartyMember{
		Name:        "Kris",
		AccentColor: color.RGBA{0, 162, 232, 255},
		MaxHP:       90,
		HP:          90,
		Attack:      10,
		Defense:     10,
		IsLeader:    true,
		Acts: []battle.ActDef{
			{Name: "Threaten", Description: "Scare the opponent"},
			{Name: "Hypnosis", Description: "Put opponent to sleep"},
		},
		BattleMiniature: krisIcon,
	}

	susie := &battle.PartyMember{
		Name:        "Susie",
		AccentColor: color.RGBA{180, 80, 220, 255},
		MaxHP:       110,
		HP:          110,
		Attack:      15,
		Defense:     8,
		IsLeader:    false,
		Acts: []battle.ActDef{
			{Name: "Rude Buster", Description: "Deal heavy damage"},
			{Name: "Intimidate", Description: "Lower opponent defense"},
		},
		BattleMiniature: susieIcon,
	}

	ralsei := &battle.PartyMember{
		Name:        "Ralsei",
		AccentColor: color.RGBA{0, 200, 100, 255},
		MaxHP:       70,
		HP:          70,
		Attack:      5,
		Defense:     15,
		IsLeader:    false,
		Acts: []battle.ActDef{
			{Name: "Heal Prayer", Description: "Restore HP", TargetSelf: true},
			{Name: "Pacify", Description: "Calm the opponent"},
		},
		BattleMiniature: ralseiIcon,
	}

	jevil := &battle.Opponent{
		Name:     "Enemy",
		MaxHP:    500,
		HP:       500,
		Attack:   12,
		Defense:  10,
		MaxMercy: 100,
		Mercy:    0,
		Acts: []battle.ActDef{
			{Name: "Tire", Description: "Tire the Enemy out"},
			{Name: "Pirouette", Description: "Dance with the Enemy"},
		},
		Reactions: map[string]battle.ActReaction{
			"Tire": {StateChange: battle.StateTired, MercyAmount: 30},
		},
	}

	testTurn := &battle.Turn{
		Sequence: []battle.TurnEvent{
			&battle.DialogueEvent{
				Emitter: jevil,
				Lines:   []string{"CHAOS, CHAOS!$e"},
			},
			&battle.AttackEvent{
				Duration: 5 * time.Second,
				Sequence: &battle.AttackSequence{},
			},
		},
	}

	gameScene := &GameScene{}
	gameScene.Battle = &battle.Battle{
		TextEngine:   g.textEngine,
		SoundPlayer:  g.soundPlayer,
		Party:        []*battle.PartyMember{kris, susie, ralsei},
		Opponents:    []*battle.Opponent{jevil},
		ActiveMember: 0,
	}
	gameScene.Battle.SetMenuSprite(menuIcon)
	gameScene.Battle.SetSoulSprite(soulIcon)
	gameScene.Battle.SetArenaHooks(showArena, hideArena)
	gameScene.Battle.SetArenaBounds(sa.ArenaInner())
	gameScene.Battle.StartTurn(testTurn, []string{
		"* A wild Enemy approaches!$f",
	})

	intro := NewIntroScene(
		gameScene.Update,
		gameScene.Draw,
		func() {
			g.scenes.Pop()
			g.scenes.Push(gameScene)
		},
	)
	g.scenes.Push(intro)

	if err := g.soundPlayer.PlaySound("battle", 1.2); err != nil {
		log.Printf("music: %v", err)
	}
}

func (g *Game) Update() error {
	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now

	g.scenes.Update(dt)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.scenes.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 960
}
