package appkit

type AppEventBus struct {
	events map[string][]EventHandler
}

// Ensure AppEventBus implements EventBus.
var _ EventBus = (*AppEventBus)(nil)

func NewEventBus() *AppEventBus {
	return &AppEventBus{
		events: make(map[string][]EventHandler),
	}
}

func (b *AppEventBus) Publish(event string) {
	if _, ok := b.events[event]; !ok {
		b.events[event] = make([]EventHandler, 0)
	}
}

func (b *AppEventBus) Subscribe(event string, handler EventHandler) {
	// Since issues with order of code execution may arise,
	// it is allowed to subscribe to events that have not been registered yet.
	if _, ok := b.events[event]; !ok {
		b.events[event] = make([]EventHandler, 0)
	}

	b.events[event] = append(b.events[event], handler)
}

func (b *AppEventBus) Trigger(event string, data interface{}) {
	for _, handler := range b.events[event] {
		handler(data)
	}
}
