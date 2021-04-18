package cqrs

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

// EventID is unique string representing an event. Should be unique per event type
// across the system. Good idea is to use prefix per entity.
type EventID string

// Event is base event, containing all the needed information for something
// that had happened in the past.
type Event struct {
	EventID       EventID     `json:"event_id" mapstructure:"event_id"`
	AggregateID   string      `json:"aggregate_id" mapstructure:"aggregate_id"`
	CreatedAt     time.Time   `json:"created_at" mapstructure:"created_at"`
	CorrelationID string      `json:"correlation_id" mapstructure:"correlation_id"`
	Data          interface{} `json:"data" mapstructure:"data"`
}

// NewEvent returns instance of an event with provided ID and data, and populates
// relevant fields from provided command.
func NewEvent(ID EventID, cmd Command, data interface{}) *Event {
	return &Event{
		EventID:       ID,
		AggregateID:   cmd.GetAggregateID(),
		CreatedAt:     time.Now().UTC(),
		CorrelationID: cmd.GetCorrelationID(),
		Data:          data,
	}
}

// EventSerializer defines operations needed for event instance marshal and unmarshal operations.
type EventSerializer interface {
	Marshal(*Event) ([]byte, error)
	Unmarshal([]byte) (*Event, error)
	MarshalData(*Event) ([]byte, error)
	UnmarshalData(EventID, []byte) (interface{}, error)
}

type eventJSONSerializer struct {
	ctors map[EventID]func() interface{}
}

// NewEventJSONSerializer returns instance of EventSerializer that is using JSON as underlying format.
func NewEventJSONSerializer() *eventJSONSerializer {
	return &eventJSONSerializer{
		ctors: make(map[EventID]func() interface{}),
	}
}

func (e *eventJSONSerializer) RegisterDataCtor(ID EventID, ctor func() interface{}) {
	e.ctors[ID] = ctor
}

func (e *eventJSONSerializer) Marshal(ev *Event) ([]byte, error) {
	return json.Marshal(ev)
}

func (e *eventJSONSerializer) Unmarshal(rawData []byte) (*Event, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal(rawData, &raw); err != nil {
		return nil, err
	}
	evID, ok := raw["event_id"]
	if !ok {
		return nil, errors.New("raw event does not contain event_id")
	}
	evIdAsStr, ok := evID.(string)
	if !ok {
		return nil, fmt.Errorf("event_id has unexpected type: %T", evID)
	}
	eventID := EventID(evIdAsStr)

	ctor, ok := e.ctors[eventID]
	if !ok {
		return nil, fmt.Errorf("unkonwn event ID: %v", eventID)
	}
	ev := &Event{
		Data: ctor(),
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:   nil,
		DecodeHook: ToTimeHookFunc(),
		Result:     ev,
	})
	if err != nil {
		return nil, err
	}
	return ev, decoder.Decode(raw)
}

func (e *eventJSONSerializer) MarshalData(ev *Event) ([]byte, error) {
	return json.Marshal(ev.Data)
}

func (e *eventJSONSerializer) UnmarshalData(eventID EventID, data []byte) (interface{}, error) {
	ctor, ok := e.ctors[eventID]
	if !ok {
		return nil, fmt.Errorf("unknown event ID: %v", eventID)
	}
	eventData := ctor()
	return eventData, json.Unmarshal(data, &eventData)
}

func ToTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339Nano, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
	}
}
