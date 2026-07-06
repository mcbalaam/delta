package game

import (
	"fmt"
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

	sa := arena.NewSquareArena(640, 380)
	sa.SetSprite(arenaIcon)
	sa.SetSpritePos(0, 0, 2) // static background, independent of arena geometry

	spriteLayer := &arena.SpriteLayer{Arena: sa, Visible: true}
	boxLayer := &arena.BoxLayer{Arena: sa, Visible: false}
	queues.DefaultQueue.ScheduleAt(spriteLayer, queues.LayerArena)
	queues.DefaultQueue.ScheduleAt(boxLayer, queues.LayerArena)
	queues.DefaultUpdateQueue.Schedule(spriteLayer)
	queues.DefaultUpdateQueue.Schedule(boxLayer)

	showArena := func() {
		boxLayer.Visible = true
		boxLayer.StartEntrance()
	}
	hideArena := func() {
		boxLayer.Visible = false
	}
	startExitArena := func() {
		boxLayer.StartExit()
	}

	kris := &battle.PartyMember{
		Name:        "Kris",
		AccentColor: color.RGBA{1, 255, 255, 255},
		MaxHP:       90,
		HP:          90,
		Attack:      10,
		Defense:     10,
		IsLeader:    true,
		Acts: []battle.Act{
			battle.NewSimpleAct("Threaten", "Scare the opponent",
				"* You raise your fists menacingly. Rudinn backs away.$e",
				func(ctx interface{}) string {
					c := ctx.(*battle.ActContext)
					if t := c.TargetOpponent(); t != nil {
						t.Mercy += 10
						t.State = battle.StateFlustered
						return t.Name + " cowers in fear!$e"
					}
					return ""
				}),
			battle.NewSimpleAct("Hypnosis", "Put opponent to sleep",
				"* A swirl of colorful lights captures Rudinn's attention.$f",
				func(ctx interface{}) string {
					c := ctx.(*battle.ActContext)
					if t := c.TargetOpponent(); t != nil {
						t.ApplyStatus(&battle.StatusEffect{
							Name: "Sleep", Duration: 3,
							Modifier: battle.StatusMod{AttackMod: 0.5, DefenseMod: 0.5, MercyMod: 1.0},
						})
						return t.Name + " feels drowsy...$e"
					}
					return ""
				}),
		},
		BattleMiniature: krisIcon,
	}
	if krisChar, err := render.NewAnimatedIconFromPath("game/media/sprites/kris", "idle"); err == nil {
		kris.CharacterSprite = krisChar
	} else {
		log.Printf("kris char sprite: %v", err)
	}

	susie := &battle.PartyMember{
		Name:        "Susie",
		AccentColor: color.RGBA{255, 0, 255, 255},
		MaxHP:       110,
		HP:          110,
		Attack:      15,
		Defense:     8,
		IsLeader:    false,
		Acts: []battle.Act{
			battle.NewSimpleAct("Rude Buster", "Deal heavy damage",
				"* Susie charges up and unleashes a powerful blow!$e",
				func(ctx interface{}) string {
					c := ctx.(*battle.ActContext)
					if t := c.TargetOpponent(); t != nil {
						t.HP -= 50
						return "Rudinn takes 50 damage!$e"
					}
					return ""
				}),
			battle.NewAttackDeltaAct("Intimidate", "Lower opponent defense",
				"* You loom over Rudinn. His confidence wavers.$f", -5, false),
		},
		BattleMiniature: susieIcon,
	}

	ralsei := &battle.PartyMember{
		Name:        "Ralsei",
		AccentColor: color.RGBA{1, 255, 0, 255},
		MaxHP:       70,
		HP:          70,
		Attack:      5,
		Defense:     15,
		IsLeader:    false,
		Acts: []battle.Act{
			battle.NewHealAct("Heal Prayer", "Restore HP",
				"* The shrine glows warmly. Kris heals!$f", 25, true),
			battle.NewSimpleAct("Pacify", "Calm the opponent",
				"* You gently soothe the opponent. They seem more at peace.$f",
				func(ctx interface{}) string {
					c := ctx.(*battle.ActContext)
					if t := c.TargetOpponent(); t != nil {
						t.Mercy += 30
						return t.Name + " seems more calm.$e"
					}
					return ""
				}),
		},
		BattleMiniature: ralseiIcon,
	}

	rudinn := &battle.Opponent{
		Name:     "Rudinn",
		MaxHP:    300,
		HP:       300,
		Attack:   10,
		Defense:  8,
		MaxMercy: 100,
		Mercy:    0,
		Acts: []battle.Act{
			battle.NewRaiseMercyAct("Talk", "Try to reason with Rudinn",
				"* You try to reason with Rudinn. He listens...$f", 20),
			battle.NewRaiseMercyAct("Compliment", "Flatter Rudinn",
				"* You give a sincere compliment. Rudinn smiles warmly.$f", 30),
		},
		Reactions: map[string]battle.ActReaction{
			"Talk":       {StateChange: battle.StateTired, MercyAmount: 20},
			"Compliment": {StateChange: battle.StateFlustered, MercyAmount: 30},
		},
	}
	if rudinnChar, err := render.NewAnimatedIconFromPath("game/media/sprites/rudinn", "idle"); err == nil {
		rudinn.CharacterSprite = rudinnChar
	} else {
		log.Printf("rudinn char sprite: %v", err)
	}

	gameScene := &GameScene{
		fade: NewScreenFade(),
	}
	gameScene.Battle = &battle.Battle{
		TextEngine:   g.textEngine,
		SoundPlayer:  g.soundPlayer,
		Party:        []*battle.PartyMember{kris, susie, ralsei},
		Opponents:    []*battle.Opponent{rudinn},
		ActiveMember: 0,
	}

	firstTurn := true
	projIcon, err := render.NewAnimatedIconFromPath("media/sprites/hp", "idle")
	if err != nil {
		log.Fatalf("projectile icon: %v", err)
	}

	gameScene.Battle.OnTurnComplete = func() {
		if firstTurn {
			firstTurn = false
		}
		enemy := gameScene.Battle.Opponents[0]
		if !enemy.Alive() {
			return
		}
		nextTurn := &battle.Turn{
			Sequence: []battle.TurnEvent{
				&battle.DialogueEvent{
					Emitter: enemy,
					Lines:   []string{"Here I come!$e"},
				},
				&battle.AttackEvent{
					Duration: 6 * time.Second,
					Sequence: battle.NewHomingAttack(projIcon, 10, 500*time.Millisecond, 250, 20, 3*time.Second),
				},
			},
		}
		gameScene.Battle.StartTurn(nextTurn, []string{
			fmt.Sprintf("* %s attacks!$f", enemy.Name),
		})
	}
	if targetIcon, err := render.NewAnimatedIconFromPath("media/sprites/target", "idle"); err == nil {
		gameScene.Battle.SetTargetIcon(targetIcon)
	} else {
		log.Printf("target icon: %v", err)
	}

	if boxIcon, err := render.NewAnimatedIconFromPath("media/sprites/dialoguebox", "idle"); err == nil {
		gameScene.Battle.SetDialogueBox(boxIcon)
	} else {
		log.Printf("dialogue box: %v", err)
	}

	gameScene.Battle.SetMenuSprite(menuIcon)
	gameScene.Battle.SetSoulSprite(soulIcon)
	if hpIcon, err := render.NewAnimatedIconFromPath("media/sprites/hp", "idle"); err == nil {
		gameScene.Battle.HPIcon = hpIcon
	} else {
		log.Printf("hp icon: %v", err)
	}
	if slashIcon, err := render.NewAnimatedIconFromPath("media/sprites/slash", "idle"); err == nil {
		gameScene.Battle.SlashIcon = slashIcon
	} else {
		log.Printf("slash icon: %v", err)
	}
	gameScene.Battle.SetArenaHooks(showArena, hideArena)
	gameScene.Battle.SetStartExitArena(startExitArena)
	gameScene.Battle.SetArenaBounds(sa.ArenaInner())

	firstTurnTurn := &battle.Turn{
		Sequence: []battle.TurnEvent{
			&battle.DialogueEvent{
				Emitter: rudinn,
				Lines:   []string{"Don't come $nany closer!$e"},
			},
			&battle.AttackEvent{
				Duration: 6 * time.Second,
				Sequence: battle.NewHomingAttack(projIcon, 10, 500*time.Millisecond, 250, 20, 3*time.Second),
			},
		},
	}
	gameScene.Battle.StartTurn(firstTurnTurn, []string{
		fmt.Sprintf("* %s blocks the way!$f", rudinn.Name),
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

	if err := g.soundPlayer.PlaySound("battle", 1); err != nil {
		log.Printf("music: %v", err)
	}
}

func RunScare() {

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
