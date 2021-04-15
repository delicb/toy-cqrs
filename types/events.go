package types

import (
	"encoding/json"
	"time"
)

// EventID is unique identifier of an event that occurred.
type EventID string

// Event is a single event shared with all services. This interface defines
// basic behavior, but has an option to store a payload that is event specific.
type Event interface {
	// ID is an unique identifier of an event, something like "user.created"
	ID() EventID

	// AggregateID is unique identifier of an entity for which event is created.
	// E.g. if event is user related, this would hold user unique identifier. This
	// value should be used as opaque string for everyone except for service that
	// is the owner of the entity, no other service should read into the meaning
	// of returned value.
	AggregateID() string

	// CorrelationID stores CorrelationID of the command that caused this event.
	CorrelationID() string

	// CreatedAt returns time when the event was created for the first time.
	CreatedAt() time.Time

	// StorePayload stores value provided within a event as a payload.
	StorePayload(interface{}) error

	// LoadPayload populates provided interface (has to be a pointer) from
	// stored payload.
	LoadPayload(interface{}) error

	// RawPayload returns event payload as is, just a slice. Caller should not modify it.
	RawPayload() []byte

	// SetRawPayload sets raw payload. Do this only if it is obtained via RawPayload.
	// Once called, provided parameter should not be modified by the caller.
	SetRawPayload([]byte)

	// Marshal serializes event in a suitable way for being transferred over the network.
	Marshal() ([]byte, error)

	// Unmarshal reconstructs event instance from raw bytes, usually one produced by Marshal.
	Unmarshal([]byte) error
}

type payload []byte

func (p payload) MarshalJSON() ([]byte, error) {
	// do not marshal, just return payload as is
	return p, nil
}

func (p *payload) UnmarshalJSON(data []byte) error {
	// just keep payload as is, do not try to unmarshal it
	// it is done in LoadPayload when needed
	*p = data
	return nil
}

type BaseEvent struct {
	Xid            EventID   `json:"event_id"`
	XaggregateID   string    `json:"aggregate_id"`
	XcorrelationID string    `json:"correlation_id"`
	XcreatedAt     time.Time `json:"created_at"`
	XPayload       payload   `json:"payload"`
}

func (e *BaseEvent) ID() EventID           { return e.Xid }
func (e *BaseEvent) AggregateID() string   { return e.XaggregateID }
func (e *BaseEvent) CorrelationID() string { return e.XcorrelationID }
func (e *BaseEvent) CreatedAt() time.Time  { return e.XcreatedAt }
func (e *BaseEvent) StorePayload(payload interface{}) error {
	d, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.XPayload = d
	return nil
}
func (e *BaseEvent) LoadPayload(to interface{}) error {
	return json.Unmarshal(e.XPayload, to)
}

func (e *BaseEvent) RawPayload() []byte        { return e.XPayload }
func (e *BaseEvent) SetRawPayload(raw []byte)  { e.XPayload = raw }
func (e *BaseEvent) Marshal() ([]byte, error)  { return json.Marshal(e) }
func (e *BaseEvent) Unmarshal(in []byte) error { return json.Unmarshal(in, e) }

func NewEvent(ID EventID, aggregateID, correlationID string, payload interface{}) (Event, error) {
	ev := &BaseEvent{
		Xid:            ID,
		XaggregateID:   aggregateID,
		XcorrelationID: correlationID,
		XcreatedAt:     time.Now().UTC(),
	}
	return ev, ev.StorePayload(payload)
}

var _ Event = &BaseEvent{}

// UnmarshalEvent creates new event instance and populates it with provided raw data.
func UnmarshalEvent(in []byte) (Event, error) {
	ev := &BaseEvent{}
	return ev, ev.Unmarshal(in)
}
