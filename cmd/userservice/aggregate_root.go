package main

// AggregateRoot is an instance of a type that is entity being managed.
// For example, in user service, User type would implement and be AggregateRoot.
type AggregateRoot interface {
	// HandleCommand generates and applies relevant events for provided command to this aggregate root.
	HandleCommand(cmd Command) error

	// GetID returns unique ID for this aggregate root.
	GetID() string

	// GetChanges returns list of new events that have been applied to this aggregate root.
	GetChanges() []*Event

	// ClearChanges deletes all new events applied to this aggregate root.
	// This should be done after save to persistent database or in case of rollback.
	ClearChanges()

	// Apply changes state of this aggregate root to reflect desired outcome described by
	// provided event. Flag new is true in case new event is being added (typically during
	// HandleCommand), and it will be false if aggregate root is being reconstructed by past,
	// already stored events.
	Apply(new bool, ev *Event) error
}

// Root is partial implementation of AggregateRoot, made to make implementation of
// concrete aggregate roots easier by embedding.
type Root struct {
	ID      string
	Changes []*Event
}

func (r *Root) GetID() string        { return r.ID }
func (r *Root) GetChanges() []*Event { return r.Changes }
func (r *Root) ClearChanges()        { r.Changes = []*Event{} }
