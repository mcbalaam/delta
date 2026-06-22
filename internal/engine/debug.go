package engine

import (
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mcbalaam/delta/internal/engine/queues"
	"github.com/mcbalaam/delta/internal/render"
)

// DebugNotice renders a temporary text overlay using the "determination" font.
// It auto-destroys after the given duration and is drawn via DefaultQueue.
type DebugNotice struct {
	display  *TextDisplay
	elapsed  time.Duration
	duration time.Duration
	done     bool
}

//	text     — the string to display
//	x, y     — screen position
//	duration — how long the text should stay visible (e.g. 1*time.Second)
//
// The text is drawn at scale 0.3 in the "determination" font for debug readability.
func ShowDebugNotice(te *TextEngine, text string, x, y float64, duration time.Duration) *DebugNotice {
	if _, exists := te.FontsLoaded["cryptoftomorrow"]; !exists {
		icon, err := render.NewAnimatedIconFromPath("media/sprites/cryptoftomorrow", " ")
		if err != nil {
			return nil
		}
		te.FontsLoaded["cryptoftomorrow"] = *icon
	}
	font := te.FontsLoaded["cryptoftomorrow"]

	td := &TextDisplay{
		Font:        font,
		Text:        strings.ToUpper(text),
		StartX:      x,
		StartY:      y,
		ScaleX:      0.3,
		ScaleY:      0.3,
		FontHeight:  24.0,
		LineSpacing: 0,
		Delay:       0,
		Instant:     true,
		CharSpacing: 0,
		CharWidth:   make(map[string]int),
		Displayed:   make([]*Glyph, 0),
	}
	td.Parse()
	td.Update(0) // build all glyphs immediately

	dn := &DebugNotice{
		display:  td,
		duration: duration,
	}

	queues.DefaultQueue.ScheduleAt(dn, queues.LayerOverlay)
	queues.DefaultUpdateQueue.Schedule(dn)

	return dn
}

// Draw renders the debug notice glyphs.
func (d *DebugNotice) Draw(s *ebiten.Image) {
	if d.done || d.display == nil {
		return
	}
	d.display.Draw(s)
}

// Update counts down the lifetime and self-destroys when expired.
func (d *DebugNotice) Update(dt time.Duration) {
	if d.done {
		return
	}
	d.elapsed += dt
	if d.elapsed >= d.duration {
		d.done = true
		if d.display != nil {
			d.display.Destroy()
			d.display = nil
		}
	}
}

func (d *DebugNotice) Destroy() {
	if d.done {
		return
	}
	d.done = true
	if d.display != nil {
		d.display.Destroy()
		d.display = nil
	}
	queues.DefaultQueue.Unschedule(d)
	queues.DefaultUpdateQueue.Unschedule(d)
}
