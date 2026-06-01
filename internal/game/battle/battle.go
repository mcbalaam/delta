package battle

import (
	"github.com/mcbalaam/delta/internal/engine"
	"github.com/mcbalaam/delta/internal/sound"
)

type BattleState int

const (
	StateIntroNarrative BattleState = iota
	StateMenuSelection
	StateEnemyDialogue
	StateEnemyAttack
)

type Battle struct {
	State          BattleState
	TextEngine     *engine.TextEngine
	SoundPlayer    *sound.SoundPlayer
	CurrentTurn    *Turn
	CurrentSession *engine.DialogueSession
}

// 1. Вызывается в самом начале нового хода
func (b *Battle) StartTurn(turn *Turn) {
	b.CurrentTurn = turn
	b.State = StateIntroNarrative

	// Запускаем нарратив. В конце intro строк обязательно должен быть тег $f
	b.CurrentSession = engine.NewDialogueSession(
		b.TextEngine,
		b.CurrentTurn.IntroNarrative,
		engine.StyleNarrative,
		b.SoundPlayer,
	)

	b.CurrentSession.OnAllComplete = func() {
		// Текст допечатался до $f и остановился. Открываем меню действий игрока.
		// Текст НЕ стираем (b.CurrentSession.Destroy() не вызываем).
		b.State = StateMenuSelection
		b.ShowActionMenus()
	}

	b.CurrentSession.Start()
}

// 2. Вызывается, когда игрок подтвердил все действия (нажал FIGHT/ACT/ITEM для всей пати)
func (b *Battle) FinishPlayerTurn() {
	// Нарратив больше не нужен, очищаем подменю и запускаем бабблы врагов
	if b.CurrentSession != nil {
		b.CurrentSession.Destroy()
	}

	b.State = StateEnemyDialogue

	// Собираем все реплики врагов для этого хода в один пул сессии
	var allCharacterLines []string
	for _, speech := range b.CurrentTurn.IntroDialogues {
		// Здесь при необходимости можно модифицировать строки в зависимости от Emitter'а
		// (например, добавлять имя или позицию баббла, если это заложено в TextStyle)
		allCharacterLines = append(allCharacterLines, speech.Lines...)
	}

	// Если у врагов нет реплик в этом ходу — сразу идем в атаку
	if len(allCharacterLines) == 0 {
		b.StartEnemyAttackPhase()
		return
	}

	// Запускаем диалог врагов (использует StyleDialogue + теги $e для ожидания Z)
	b.CurrentSession = engine.NewDialogueSession(
		b.TextEngine,
		allCharacterLines,
		engine.StyleDialogue,
		b.SoundPlayer,
	)

	b.CurrentSession.OnAllComplete = func() {
		// Игрок прочитал все реплики (прожал Z на последней), теперь очищаем экран и бьем
		b.CurrentSession.Destroy()
		b.StartEnemyAttackPhase()
	}

	b.CurrentSession.Start()
}

func (b *Battle) StartEnemyAttackPhase() {
	b.State = StateEnemyAttack
	// Логика включения арены и спавна прожектайлов на основе b.CurrentTurn.AttackPatternID
}

func (b *Battle) ShowActionMenus() {
	// Включение рендеринга кнопок FIGHT/ACT
}
