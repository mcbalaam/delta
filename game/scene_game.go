package game

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/pkg/battle"
)

type GameScene struct {
	Battle        *battle.Battle
	dialogueBoxID interface{}
	fade          *ScreenFade
}

func (s *GameScene) Update(dt time.Duration) {
	if s.Battle != nil {
		s.Battle.Update(dt)
		s.Battle.NavigateMenu()
	}
	queues.DefaultUpdateQueue.Execute(dt)
	queues.DefaultDeleteQueue.Execute()

	if s.Battle != nil {
		s.Battle.ScheduleDialogueBox(&s.dialogueBoxID)
	}
}

func (s *GameScene) Draw(screen *ebiten.Image) {
	queues.DefaultQueue.Execute(screen)
	if s.Battle != nil {
		s.Battle.DrawMenu(screen)
	}
}

// ── ScreenFade ───────────────────────────────────────────────────

type ScreenFade struct {
	elapsed time.Duration
	done    bool
	img     *ebiten.Image
}

func NewScreenFade() *ScreenFade {
	f := &ScreenFade{
		img: ebiten.NewImage(1280, 960),
	}
	queues.DefaultQueue.ScheduleAt(f, -150)
	queues.DefaultUpdateQueue.Schedule(f)
	return f
}

func (f *ScreenFade) Update(dt time.Duration) {
	f.elapsed += dt
}

func (f *ScreenFade) Draw(screen *ebiten.Image) {
	if f.done {
		return
	}

	t := f.elapsed.Seconds()

	switch {
	case t < 16:
		f.img.Fill(color.Black)
		screen.DrawImage(f.img, nil)

	case t < 16.5:
		p := (t - 16) / 0.5
		c := lerpColor(color.Black, color.White, p)
		f.img.Fill(c)
		screen.DrawImage(f.img, nil)

	case t < 17:
		p := float32((t - 16.5) / 0.5)
		f.img.Fill(color.White)
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(1 - p)
		screen.DrawImage(f.img, op)

	default:
		f.done = true
		return
	}
}

func lerpColor(a, b color.Color, t float64) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	return color.RGBA{
		R: uint8((float64(ar>>8)*(1-t) + float64(br>>8)*t) / 1),
		G: uint8((float64(ag>>8)*(1-t) + float64(bg>>8)*t) / 1),
		B: uint8((float64(ab>>8)*(1-t) + float64(bb>>8)*t) / 1),
		A: uint8(aa >> 8),
	}
}
