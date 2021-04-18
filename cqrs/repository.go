package cqrs

import (
	"fmt"
)

// Repository manages aggregate root objects.
type Repository interface {
	// Load creates and returns aggregate root with provided id.
	Load(typ, aggregateID string) (AggregateRoot, error)

	// Save stores new events from provided aggregate root.
	Save(root AggregateRoot) error
}

// AggregateRootCtor is a function that returns empty instance of an aggregate root.
type AggregateRootCtor func() AggregateRoot

type simpleRepository struct {
	ctors map[string]AggregateRootCtor
	store EventStore
}

func (r *simpleRepository) Load(typ, aggregateID string) (AggregateRoot, error) {
	ctor, ok := r.ctors[typ]
	if !ok {
		return nil, fmt.Errorf("unknown aggregate type: %v", typ)
	}
	root := ctor()

	// if we did not get aggregate ID, this might be one of the root events (e.g. create),
	// so there is no much sense to query the DB, just return empty aggregate root
	if aggregateID == "" {
		return root, nil
	}

	// otherwise load old events and apply them
	oldEvents, err := r.store.Load(aggregateID)
	if err != nil {
		return nil, err
	}

	for _, ev := range oldEvents {
		if err := root.Apply(false, ev); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func (r *simpleRepository) Save(root AggregateRoot) error {
	err := r.store.Save(root.GetChanges())
	if err != nil {
		return err
	}
	root.ClearChanges()
	return nil
}

func (r *simpleRepository) RegisterCtor(typ string, ctor AggregateRootCtor) {
	r.ctors[typ] = ctor
}

func NewSimpleRepository(store EventStore) *simpleRepository {
	return &simpleRepository{
		ctors: make(map[string]AggregateRootCtor),
		store: store,
	}
}
