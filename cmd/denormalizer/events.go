package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type EventID string

const UserCreatedID EventID = "user.created"
const EmailChangedID EventID = "user.email.changed"
const PasswordChangedID EventID = "user.password.changed"
const EnabledID EventID = "user.enabled"
const DisabledID EventID = "user.disabled"

type payload []byte

func (p payload) MarshalJSON() ([]byte, error)     { return p, nil }
func (p *payload) UnmarshalJSON(data []byte) error { *p = data; return nil }

// Event is base event, containing all the needed information for something
// that had happened in the past.Ò
type Event struct {
	EventID       EventID   `json:"event_id,omitempty"`
	AggregateID   string    `json:"aggregate_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Data          interface{}
	Payload       payload `json:"payload,omitempty"`
}

// UserCreated is event indicating that new user has been created.
type UserCreated struct {
	ID        string `json:"id,omitempty"`
	Email     string `json:"email,omitempty"`
	Password  string `json:"password,omitempty"`
	IsEnabled bool   `json:"is_enabled,omitempty"`
}

// UserEmailChanged is event indicating that user's email has been changed.
type UserEmailChanged struct {
	NewEmail string `json:"new_email,omitempty"`
	OldEmail string `json:"old_email,omitempty"`
}

// UserPasswordChanged is event indicating that user's password has been changed.
type UserPasswordChanged struct {
	NewPassword string `json:"new_password,omitempty"`
	OldPassword string `json:"old_password,omitempty"`
}

// UserEnabled is event indicating that user has been enabled.
type UserEnabled struct{}

// UserDisabled is event indicating that user has been disabled.
type UserDisabled struct{}

// UnmarshalEvent returns Event instance from raw bytes provided.
func UnmarshalEvent(data []byte) (*Event, error) {
	ev := &Event{}
	if err := json.Unmarshal(data, ev); err != nil {
		return nil, err
	}
	var parsedData interface{}
	switch ev.EventID {
	case "user.created":
		parsedData = &UserCreated{}
	case "user.email.changed":
		parsedData = &UserEmailChanged{}
	case "user.password.changed":
		parsedData = &UserPasswordChanged{}
	case "user.enabled":
		parsedData = &UserEnabled{}
	case "user.disabled":
		parsedData = &UserDisabled{}
	default:
		return nil, fmt.Errorf("unknown event while unmarshaling: %v", ev.EventID)
	}

	if err := json.Unmarshal(ev.Payload, parsedData); err != nil {
		return nil, err
	}
	ev.Data = parsedData
	return ev, nil
}
