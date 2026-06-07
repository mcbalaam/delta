package game

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
	"github.com/mcbalaam/delta/pkg/arena"
	"github.com/mcbalaam/delta/pkg/background"
	"github.com/mcbalaam/delta/pkg/battle"
)

type Game struct {
	last          time.Time
	soundPlayer   *sound.SoundPlayer
	textEngine    *engine.TextEngine
	CurrentBattle *battle.Battle
	layoutOverlay *ebiten.Image // debug layout reference
}

func NewGame(soundPlayer *sound.SoundPlayer, textEngine *engine.TextEngine) *Game {
	g := &Game{
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

	// ── Background grid ──
	bgGrid := background.NewDualGridBG()
	engine.DefaultQueue.Schedule(&background.GridLayer{BG: bgGrid})
	engine.DefaultUpdateQueue.Schedule(&background.GridLayer{BG: bgGrid})

	// ── Arena box (hidden until enemy turn) ──
	sa := arena.NewSquareArena(640, 480)
	sa.SetSprite(arenaIcon)

	spriteLayer := &arena.SpriteLayer{Arena: sa, Visible: true}
	boxLayer := &arena.BoxLayer{Arena: sa, Visible: false}
	engine.DefaultQueue.Schedule(spriteLayer)
	engine.DefaultQueue.Schedule(boxLayer)
	engine.DefaultUpdateQueue.Schedule(spriteLayer)

	showArena := func() {
		boxLayer.Visible = true
	}
	hideArena := func() {
		boxLayer.Visible = false
	}

	// ── Party members ──
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

	// ── Opponent ──
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
			//			"Pirouette":  {MercyAmount: 15},
			//			"Threaten":   {MercyAmount: 10},
			//			"Intimidate": {MercyAmount: 10, AttackDelta: -2},
			//			"Pacify":     {MercyAmount: 25, StateChange: battle.StateFlustered},
			//			"Hypnosis":   {MercyAmount: 20},
		},
	}

	// ── Turn script ──
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

	// ── Battle instance ──
	b := &battle.Battle{
		TextEngine:   g.textEngine,
		SoundPlayer:  g.soundPlayer,
		Party:        []*battle.PartyMember{kris, susie, ralsei},
		Opponents:    []*battle.Opponent{jevil},
		ActiveMember: 0,
	}
	b.SetMenuSprite(menuIcon)
	b.SetSoulSprite(soulIcon)
	b.SetArenaHooks(showArena, hideArena)
	b.StartTurn(testTurn, []string{
		"* A wild Enemy approaches!$f",
	})

	g.CurrentBattle = b
}

func (g *Game) Update() error {
	now := time.Now()
	if g.last.IsZero() {
		g.last = now
	}
	dt := now.Sub(g.last)
	g.last = now

	if g.CurrentBattle != nil {
		g.CurrentBattle.Update(dt)
		g.CurrentBattle.NavigateMenu()
	}

	engine.DefaultUpdateQueue.Execute(dt)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	engine.DefaultQueue.Execute(screen)

	if g.CurrentBattle != nil {
		g.CurrentBattle.DrawMenu(screen)
	}

	// // Debug layout overlay at 30% opacity
	// if g.layoutOverlay != nil {
	// 	op := &ebiten.DrawImageOptions{}
	// 	op.ColorScale.ScaleAlpha(0.3)
	// 	op.GeoM.Scale(2.0, 2.0)
	// 	screen.DrawImage(g.layoutOverlay, op)
	// }
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 960
}
