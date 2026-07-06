package battle

import (
	"github.com/mcbalaam/delta/internal/engine"
)

// ── Narrative helpers ────────────────────────────────────────────

func (b *Battle) clearNarrativeText() {
	if b.turnSession != nil {
		b.turnSession.Destroy()
		b.turnSession = nil
	}
	if b.restoredText != nil {
		b.restoredText.Destroy()
		b.restoredText = nil
	}
}

func (b *Battle) restoreNarrative() {
	if len(b.narrativeLines) == 0 {
		return
	}
	if b.restoredText != nil {
		b.restoredText.Destroy()
		b.restoredText = nil
	}

	lastLine := b.narrativeLines[len(b.narrativeLines)-1]
	td, _ := b.TextEngine.DisplayText(
		engine.StyleNarrative.WithInstant(true),
		lastLine,
		b.SoundPlayer,
		nil,
	)
	b.restoredText = td
}

func (b *Battle) showActionNarrative(text string, onDone func()) {
	b.clearNarrativeText()
	session := engine.NewDialogueSession(
		b.TextEngine,
		[]string{text},
		engine.StyleNarrative,
		b.SoundPlayer,
	)
	session.OnAllComplete = func() {
		session.Destroy()
		if onDone != nil {
			onDone()
		}
	}
	b.turnSession = session
	session.Start()
}
