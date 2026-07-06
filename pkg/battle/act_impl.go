package battle

// Act is the interface for a single ACT action in battle.
// Each ACT has a name, description, narration, and an Execute method
// that performs the action on the battle context.
type Act interface {
	Name() string
	Description() string
	Narration() string
	Execute(battleCtx interface{}) string
}

// ActDef describes a unique ACT action available for this opponent.
// This is kept for backwards compatibility only.
type ActDef struct {
	Name        string
	Description string
	TargetSelf  bool
	OnRun       func()
}

// ActContext provides the battle state to an ACT's Execute method.
type ActContext struct {
	Battle       *Battle
	ActiveMember *PartyMember
	Target       interface{} // *PartyMember or *Opponent
	TargetIdx    int
	IsAllyTarget bool
}

func (ctx *ActContext) TargetPartyMember() *PartyMember {
	if t, ok := ctx.Target.(*PartyMember); ok {
		return t
	}
	return nil
}

func (ctx *ActContext) TargetOpponent() *Opponent {
	if t, ok := ctx.Target.(*Opponent); ok {
		return t
	}
	return nil
}

// ── Pre-built ACT helpers ──────────────────────────────────────────

type healAct struct {
	name        string
	description string
	narration   string
	amount      float64
	targetSelf  bool
}

func (a *healAct) Name() string                          { return a.name }
func (a *healAct) Description() string                   { return a.description }
func (a *healAct) Narration() string                     { return a.narration }
func (a *healAct) Execute(battleCtx interface{}) string {
	ctx := battleCtx.(*ActContext)
	var target *PartyMember
	if a.targetSelf {
		target = ctx.ActiveMember
	} else if t := ctx.TargetPartyMember(); t != nil {
		target = t
	}
	if target != nil {
		target.HP += a.amount
		if target.HP > target.MaxHP {
			target.HP = target.MaxHP
		}
	}
	return a.narration
}

type raiseMercyAct struct {
	name        string
	description string
	narration   string
	amount      float64
}

func (a *raiseMercyAct) Name() string        { return a.name }
func (a *raiseMercyAct) Description() string { return a.description }
func (a *raiseMercyAct) Narration() string   { return a.narration }
func (a *raiseMercyAct) Execute(battleCtx interface{}) string {
	ctx := battleCtx.(*ActContext)
	if t := ctx.TargetOpponent(); t != nil {
		t.Mercy += a.amount
	}
	return a.narration
}

type stateChangeAct struct {
	name        string
	description string
	narration   string
	state       OpponentState
}

func (a *stateChangeAct) Name() string        { return a.name }
func (a *stateChangeAct) Description() string { return a.description }
func (a *stateChangeAct) Narration() string   { return a.narration }
func (a *stateChangeAct) Execute(battleCtx interface{}) string {
	ctx := battleCtx.(*ActContext)
	if t := ctx.TargetOpponent(); t != nil {
		t.State = a.state
	}
	return a.narration
}

type attackDeltaAct struct {
	name        string
	description string
	narration   string
	delta       float64
	targetSelf  bool
}

func (a *attackDeltaAct) Name() string        { return a.name }
func (a *attackDeltaAct) Description() string { return a.description }
func (a *attackDeltaAct) Narration() string   { return a.narration }
func (a *attackDeltaAct) Execute(battleCtx interface{}) string {
	ctx := battleCtx.(*ActContext)
	var target *PartyMember
	if a.targetSelf {
		target = ctx.ActiveMember
	} else if t := ctx.TargetPartyMember(); t != nil {
		target = t
	}
	if target != nil {
		target.Attack += a.delta
	}
	return a.narration
}

type statusAct struct {
	name        string
	description string
	narration   string
	status      *StatusEffect
	targetSelf  bool
}

func (a *statusAct) Name() string        { return a.name }
func (a *statusAct) Description() string { return a.description }
func (a *statusAct) Narration() string   { return a.narration }
func (a *statusAct) Execute(battleCtx interface{}) string {
	ctx := battleCtx.(*ActContext)
	var target interface{}
	if a.targetSelf {
		target = ctx.ActiveMember
	} else if ctx.IsAllyTarget {
		target = ctx.TargetPartyMember()
	} else {
		target = ctx.TargetOpponent()
	}
	if target != nil {
		switch t := target.(type) {
		case *PartyMember:
			t.ApplyStatus(a.status)
		case *Opponent:
			t.ApplyStatus(a.status)
		}
	}
	return a.narration
}

type simpleAct struct {
	name        string
	description string
	narration   string
	execute     func(interface{}) string
}

func (a *simpleAct) Name() string                          { return a.name }
func (a *simpleAct) Description() string                   { return a.description }
func (a *simpleAct) Narration() string                     { return a.narration }
func (a *simpleAct) Execute(battleCtx interface{}) string {
	if a.execute != nil {
		return a.execute(battleCtx)
	}
	return a.narration
}

type comboAct struct {
	name        string
	description string
	narration   string
	acts        []Act
}

func (a *comboAct) Name() string        { return a.name }
func (a *comboAct) Description() string { return a.description }
func (a *comboAct) Narration() string   { return a.narration }
func (a *comboAct) Execute(battleCtx interface{}) string {
	var result string
	for _, act := range a.acts {
		r := act.Execute(battleCtx)
		if r != "" {
			result = r
		}
	}
	return result
}

// ── Constructors ───────────────────────────────────────────────────

func NewHealAct(name, description, narration string, amount float64, targetSelf bool) Act {
	return &healAct{
		name:        name,
		description: description,
		narration:   narration,
		amount:      amount,
		targetSelf:  targetSelf,
	}
}

func NewRaiseMercyAct(name, description, narration string, amount float64) Act {
	return &raiseMercyAct{
		name:        name,
		description: description,
		narration:   narration,
		amount:      amount,
	}
}

func NewStateChangeAct(name, description, narration string, state OpponentState) Act {
	return &stateChangeAct{
		name:        name,
		description: description,
		narration:   narration,
		state:       state,
	}
}

func NewAttackDeltaAct(name, description, narration string, delta float64, targetSelf bool) Act {
	return &attackDeltaAct{
		name:        name,
		description: description,
		narration:   narration,
		delta:       delta,
		targetSelf:  targetSelf,
	}
}

func NewStatusAct(name, description, narration string, status *StatusEffect, targetSelf bool) Act {
	return &statusAct{
		name:        name,
		description: description,
		narration:   narration,
		status:      status,
		targetSelf:  targetSelf,
	}
}

func NewSimpleAct(name, description, narration string, execute func(interface{}) string) Act {
	return &simpleAct{
		name:        name,
		description: description,
		narration:   narration,
		execute:     execute,
	}
}

func NewComboAct(name, description, narration string, acts []Act) Act {
	return &comboAct{
		name:        name,
		description: description,
		narration:   narration,
		acts:        acts,
	}
}
