package systems

type SignalInteractor interface{}

type Signal struct {
	Name    string
	Emitent *SignalInteractor
}

type SignalHandler func(Signal)

type SignalSubscription struct {
	Name      string
	Recepient *SignalInteractor
	Handler   SignalHandler
}

type SignalBus struct {
	Subscriptions []SignalSubscription
}

var MasterSignalBus = SignalBus{}

func (b *SignalBus) Subscribe(name string, recepient SignalInteractor, handler SignalHandler) {
	b.Subscriptions = append(b.Subscriptions, SignalSubscription{
		Name:      name,
		Recepient: &recepient,
		Handler:   handler,
	})
}

func (b *SignalBus) Emit(name string, source SignalInteractor) {
	signal := Signal{
		Name:    name,
		Emitent: &source,
	}

	for _, sub := range b.Subscriptions {
		if sub.Name == name {
			sub.Handler(signal)
		}
	}
}
