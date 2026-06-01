package engine

import (
	"image/color"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	Color  color.Color
}

func (g *Glyph) Draw(s *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest

	op.GeoM.Scale(g.ScaleX, g.ScaleY)
	op.GeoM.Rotate(g.Tilt)

	op.GeoM.Translate(math.Round(g.PosX), math.Round(g.PosY))

	op.ColorScale.ScaleWithColor(g.Color)
	s.DrawImage(g.Image, op)
}

type CommandType int

const (
	CmdChar CommandType = iota
	CmdEnd
	CmdEndNoWait
)

type DialogueCommand struct {
	Type      CommandType
	Char      string
	Color     color.Color
	X         float64
	Y         float64
	TriggerAt float64
}

type TextDisplay struct {
	Font        render.AnimatedIcon
	Text        string
	StartX      float64
	StartY      float64
	ScaleX      float64
	ScaleY      float64
	FontHeight  float64
	LineSpacing float64
	Tilt        float64
	Delay       float64
	ElapsedTime float64
	IsComplete  bool
	CharWidth   map[string]int
	SoundPlayer *sound.SoundPlayer
	Instant     bool

	Commands    []DialogueCommand
	CmdIndex    int
	Displayed   []*Glyph
	OnComplete  func()
	WaitingForZ bool
}

func (t *TextDisplay) Parse() {
	runes := []rune(t.Text)
	var cmds []DialogueCommand

	currentTime := 0.0
	curX := t.StartX
	curY := t.StartY
	curColor := color.Color(color.White)
	curDelay := t.Delay

	i := 0
	for i < len(runes) {
		if runes[i] == '$' && i+1 < len(runes) {
			cmdChar := runes[i+1]
			i += 2

			switch cmdChar {
			case 'p':
				valStr := ""
				for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
					valStr += string(runes[i])
					i++
				}
				ms, _ := strconv.Atoi(valStr)
				currentTime += float64(ms) / 1000.0

			case 'c':
				if i+6 <= len(runes) {
					hexStr := string(runes[i : i+6])
					i += 6
					r, _ := strconv.ParseUint(hexStr[0:2], 16, 8)
					g, _ := strconv.ParseUint(hexStr[2:4], 16, 8)
					b, _ := strconv.ParseUint(hexStr[4:6], 16, 8)
					curColor = color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
				}

			case 's':
				valStr := ""
				for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
					valStr += string(runes[i])
					i++
				}
				ms, _ := strconv.Atoi(valStr)
				curDelay = float64(ms) / 1000.0

			case 'n':
				curX = t.StartX
				curY += (t.FontHeight + t.LineSpacing) * t.ScaleY

			case 'e':
				cmds = append(cmds, DialogueCommand{
					Type:      CmdEnd,
					TriggerAt: currentTime,
				})
			case 'f':
				cmds = append(cmds, DialogueCommand{
					Type:      CmdEndNoWait,
					TriggerAt: currentTime,
				})
			}
			continue
		}

		char := string(runes[i])
		charWidth := t.CharWidth[char]
		if charWidth == 0 {
			charWidth = 20 // Дефолтное значение для 1:1 масштаба
		}

		extraDelay := 0.0
		if char == "." || char == "," {
			extraDelay = curDelay * 10
		}

		cmds = append(cmds, DialogueCommand{
			Type:      CmdChar,
			Char:      char,
			Color:     curColor,
			X:         curX,
			Y:         curY,
			TriggerAt: currentTime,
		})

		currentTime += curDelay + extraDelay
		curX += ((float64(charWidth) + 2) * 3) * t.ScaleX
		i++
	}

	t.Commands = cmds
}

func (t *TextDisplay) Update(deltaTime time.Duration) {
	if t.IsComplete {
		return
	}

	// Мгновенная сборка всей строки
	if t.Instant {
		for t.CmdIndex < len(t.Commands) {
			cmd := t.Commands[t.CmdIndex]
			if cmd.Type == CmdEnd {
				t.WaitingForZ = true
				t.CmdIndex++
				return
			}
			if cmd.Type == CmdEndNoWait {
				t.IsComplete = true
				t.CmdIndex++
				if t.OnComplete != nil {
					t.OnComplete()
				}
				return
			}
			if cmd.Type == CmdChar {
				t.Font.SetIconState(cmd.Char)
				glyph := &Glyph{
					Image:  t.Font.CurrentState.CurrentFrameRef.Image,
					PosX:   cmd.X,
					PosY:   cmd.Y,
					ScaleX: t.ScaleX, // Чистый игровой масштаб
					ScaleY: t.ScaleY,
					Tilt:   t.Tilt,
					Color:  cmd.Color,
				}
				t.Displayed = append(t.Displayed, glyph)
			}
			t.CmdIndex++
		}
		t.IsComplete = true
		if t.OnComplete != nil {
			t.OnComplete()
		}
		return
	}

	if t.WaitingForZ {
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			t.WaitingForZ = false
			t.IsComplete = true
			if t.OnComplete != nil {
				t.OnComplete()
			}
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		for i := t.CmdIndex; i < len(t.Commands); i++ {
			if t.Commands[i].Type == CmdChar {
				t.Commands[i].TriggerAt = 0
			}
		}
		t.ElapsedTime = 100000.0
	} else {
		t.ElapsedTime += deltaTime.Seconds()
	}

	for t.CmdIndex < len(t.Commands) {
		cmd := t.Commands[t.CmdIndex]
		if cmd.TriggerAt > t.ElapsedTime {
			break
		}

		if cmd.Type == CmdEnd {
			t.WaitingForZ = true
			t.CmdIndex++
			return
		}

		if cmd.Type == CmdEndNoWait {
			t.IsComplete = true
			t.CmdIndex++
			if t.OnComplete != nil {
				t.OnComplete()
			}
			return
		}

		if cmd.Type == CmdChar {
			t.Font.SetIconState(cmd.Char)

			glyph := &Glyph{
				Image:  t.Font.CurrentState.CurrentFrameRef.Image,
				PosX:   cmd.X,
				PosY:   cmd.Y,
				ScaleX: t.ScaleX,
				ScaleY: t.ScaleY,
				Tilt:   t.Tilt,
				Color:  cmd.Color,
			}

			t.Displayed = append(t.Displayed, glyph)

			if t.SoundPlayer != nil && !inpututil.IsKeyJustPressed(ebiten.KeyX) {
				if err := t.SoundPlayer.PlayVariable("snd_text", 0.9, 0.1); err != nil {
					log.Printf("Error playing sound: %v", err)
				}
			}
		}

		t.CmdIndex++
	}

	if t.CmdIndex >= len(t.Commands) && !t.WaitingForZ {
		t.IsComplete = true
		if t.OnComplete != nil {
			t.OnComplete()
		}
	}
}

func (t *TextDisplay) Draw(s *ebiten.Image) {
	for _, glyph := range t.Displayed {
		glyph.Draw(s)
	}
}

func (t *TextDisplay) Destroy() {
	t.Displayed = nil
	t.Commands = nil
}

func (te *TextEngine) DisplayText(style TextStyle, text string,
	soundPlayer *sound.SoundPlayer, onComplete func()) (*TextDisplay, error) {

	if _, exists := te.FontsLoaded[style.FontName]; !exists {
		icon, err := render.NewAnimatedIconFromPath("media/sprites/"+style.FontName, " ")
		if err != nil {
			return nil, err
		}
		te.FontsLoaded[style.FontName] = *icon
	}

	font := te.FontsLoaded[style.FontName]

	textDisplay := &TextDisplay{
		Font:        font,
		Text:        text,
		StartX:      style.StartX,
		StartY:      style.StartY,
		ScaleX:      style.ScaleX,
		ScaleY:      style.ScaleY,
		FontHeight:  style.FontHeight,
		LineSpacing: style.LineSpacing,
		Delay:       style.DefaultDelay,
		IsComplete:  false,
		CharWidth:   make(map[string]int),
		SoundPlayer: soundPlayer,
		Displayed:   make([]*Glyph, 0),
		OnComplete:  onComplete,
		Instant:     style.Instant,
	}

	textDisplay.Parse()

	DefaultQueue.Schedule(textDisplay)
	DefaultUpdateQueue.Schedule(textDisplay)

	return textDisplay, nil
}
