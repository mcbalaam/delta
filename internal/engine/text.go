package engine

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/render"
	"github.com/mcbalaam/delta/internal/sound"
)

type TextEngine struct {
	FontsLoaded map[string]render.AnimatedIcon
}

type Glyph struct {
	Image  *ebiten.Image
	PosX   float64
	PosY   float64
	ScaleX float64
	ScaleY float64
	Tilt   float64
}

func (g *Glyph) Draw(s *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(g.ScaleX, g.ScaleY)
	op.GeoM.Rotate(g.Tilt)
	op.GeoM.Translate(g.PosX, g.PosY)
	s.DrawImage(g.Image, op)
}

type TextDisplay struct {
	Font         render.AnimatedIcon
	Text         string
	StartX       float64
	StartY       float64
	ScaleX       float64
	ScaleY       float64
	Tilt         float64
	Delay        float64
	ElapsedTime  float64
	CurrentIndex int
	IsComplete   bool
	CharWidth    map[string]int
	SoundPlayer  *sound.SoundPlayer
}

func (t *TextDisplay) Update(deltaTime time.Duration) {
	if t.IsComplete {
		return
	}

	if t.Delay <= 0 {
		return
	}

	t.ElapsedTime += deltaTime.Seconds()

	nextIndex := -1
	for i := 0; i < len(t.Text); i++ {
		if t.GetLogicalTimeForIndex(i) <= t.ElapsedTime {
			nextIndex = i
		} else {
			break
		}
	}

	if nextIndex < 0 {
		nextIndex = 0
	}

	if nextIndex >= len(t.Text)-1 {
		nextIndex = len(t.Text) - 1
		t.IsComplete = true
	}

	if nextIndex != t.CurrentIndex {
		t.CurrentIndex = nextIndex
		char := string(t.Text[t.CurrentIndex])
		t.Font.SetIconState(char)

		if t.SoundPlayer != nil {
			if err := t.SoundPlayer.PlayVariable("snd_text", 0.5, 0.1); err != nil {
				log.Printf("Error playing sound: %v", err)
			}
		}
	}
}

func (t *TextDisplay) GetLogicalTimeForIndex(index int) float64 {
	if index < 0 {
		return 0
	}
	if index >= len(t.Text) {
		index = len(t.Text) - 1
	}

	totalTime := 0.0
	for i := 0; i <= index; i++ {
		totalTime += t.Delay

		if i > 0 && string(t.Text[i-1]) == "," {
			totalTime += 10 * t.Delay
		}

		if i > 0 && string(t.Text[i-1]) == "." {
			totalTime += 10 * t.Delay
		}
	}
	return totalTime
}

func (t *TextDisplay) Draw(s *ebiten.Image) {
	posX := t.StartX

	for i := 0; i <= t.CurrentIndex && i < len(t.Text); i++ {
		char := string(t.Text[i])
		t.Font.SetIconState(char)

		charWidth := t.CharWidth[char]
		if charWidth == 0 {
			charWidth = 20
		}

		glyph := &Glyph{
			Image:  t.Font.CurrentState.CurrentFrameRef.Image,
			PosX:   posX,
			PosY:   t.StartY,
			ScaleX: t.ScaleX,
			ScaleY: t.ScaleY,
			Tilt:   t.Tilt,
		}
		glyph.Draw(s)

		posX += float64(charWidth) + 2
	}
}

func (te *TextEngine) DisplayText(fontName string, startX float64, startY float64,
	scaleX float64, scaleY float64, tilt float64, text string, delaySeconds float64, soundPlayer *sound.SoundPlayer) (*TextDisplay, error) {

	if _, exists := te.FontsLoaded[fontName]; !exists {
		icon, err := render.NewAnimatedIconFromPath("media/sprites/"+fontName, " ")
		if err != nil {
			return nil, err
		}
		te.FontsLoaded[fontName] = *icon
	}

	font := te.FontsLoaded[fontName]

	charWidths := make(map[string]int)

	textDisplay := &TextDisplay{
		Font:         font,
		Text:         text,
		StartX:       startX,
		StartY:       startY,
		ScaleX:       scaleX,
		ScaleY:       scaleY,
		Tilt:         tilt,
		Delay:        delaySeconds,
		CurrentIndex: 0,
		IsComplete:   false,
		CharWidth:    charWidths,
		SoundPlayer:  soundPlayer,
	}

	DefaultQueue.Schedule(textDisplay)
	DefaultUpdateQueue.Schedule(textDisplay)

	return textDisplay, nil
}
