package cqrs

// EventStore is description of persistence for events.
type EventStore interface {
	// Load returns all events for provided aggregate root id.
	Load(aggregateID string) ([]*Event, error)

	// Save persist all provided events.
	Save([]*Event) error
}

// EventHook is a function to be called during event lifecycle
type EventHook func(*Event)

// inMemoryStore is simple implementation of storage that does not persist
// events, but keeps them in memory instead.
type inMemoryStore struct {
	state          map[string][]*Event
	afterSaveHooks []EventHook
}

func (s *inMemoryStore) Load(aggregateID string) ([]*Event, error) {
	return s.state[aggregateID], nil
}

func (s *inMemoryStore) Save(events []*Event) error {
	for _, ev := range events {
		s.state[ev.AggregateID] = append(s.state[ev.AggregateID], ev)
		for _, h := range s.afterSaveHooks {
			h(ev)
		}
	}

	return nil
}

func (s *inMemoryStore) AddAfterSaveHook(h EventHook) {
	s.afterSaveHooks = append(s.afterSaveHooks, h)
}

// NewInMemoryEventStore returns EventStore implementation that stores events only in memory.
func NewInMemoryEventStore() *inMemoryStore {
	return &inMemoryStore{
		state:          make(map[string][]*Event),
		afterSaveHooks: make([]EventHook, 0),
	}
}
