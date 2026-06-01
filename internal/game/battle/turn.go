package battle

//import (
//	"github.com/mcbalaam/delta/internal/engine"
//	"github.com/mcbalaam/delta/internal/sound"
//)

// DialogueEmitter — интерфейс для участников боя из прошлого обсуждения
type DialogueEmitter interface {
	GetDialogueName() string
}

type CharacterSpeech struct {
	Emitter DialogueEmitter
	Lines   []string // Реплики конкретного врага (например: ["Ты не пройдешь!$e"])
}

type Turn struct {
	// 1. Фаза описания: текст в начале хода (использует StyleNarrative + в конце строк должен быть $f)
	IntroNarrative []string

	// 2. Фаза реплик: перед тем как начнется bullet-hell (использует StyleDialogue + в конце строк $e)
	IntroDialogues []CharacterSpeech

	// 3. Данные для фазы атаки (патерны прожектайлов, длительность и т.д.)
	AttackPatternID string
	AttackDuration  float64
}
