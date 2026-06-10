package battle

// ActSource identifies where an act came from — used for resolving effects.
type ActSource int

const (
	ActSourceMember   ActSource = iota // from the currently active party member
	ActSourceOpponent                  // from the target opponent
)

// ActEntry is a single entry in the combined ACT menu.
type ActEntry struct {
	Def     ActDef
	Source  ActSource
	Emitter interface{} // *PartyMember or *Opponent — origin for effect lookup
}

// CollectActs builds the ACT menu for the current member.
//
// Order (highest priority first):
//  1. targetOpponent.Acts (opponent-specific acts)
//  2. activeMember.Acts   (current party member's personal acts)
//
// Duplicates by Name are resolved in favour of the member's acts.
// Order is stable: entries appear in the order they were first encountered.
func CollectActs(activeMember *PartyMember, targetOpponent *Opponent) []ActEntry {
	seen := make(map[string]ActEntry)
	var order []string // preserve insertion order

	add := func(source ActSource, emitter interface{}, defs []ActDef) {
		for _, a := range defs {
			if _, exists := seen[a.Name]; !exists {
				order = append(order, a.Name)
			}
			seen[a.Name] = ActEntry{Def: a, Source: source, Emitter: emitter}
		}
	}

	// Opponent acts first, so member acts overwrite on name collision.
	if targetOpponent != nil {
		add(ActSourceOpponent, targetOpponent, targetOpponent.Acts)
	}
	if activeMember != nil {
		add(ActSourceMember, activeMember, activeMember.Acts)
	}

	out := make([]ActEntry, 0, len(order))
	for _, name := range order {
		out = append(out, seen[name])
	}
	return out
}
