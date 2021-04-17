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

// Event is base event, containing all the needed information for something
// that had happened in the past.
type Event struct {
	EventID       EventID
	AggregateID   string
	CreatedAt     time.Time
	CorrelationID string
	Data          interface{}
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

// UnmarshalEventData deserializes raw JSON data into type appropriate for provided eventID.
func UnmarshalEventData(eventID EventID, data []byte) (interface{}, error) {
	var inst interface{}
	switch eventID {
	case UserCreatedID:
		inst = &UserCreated{}
	case EmailChangedID:
		inst = &UserEmailChanged{}
	case PasswordChangedID:
		inst = &UserPasswordChanged{}
	case EnabledID:
		inst = &UserEnabled{}
	case DisabledID:
		inst = &UserDisabled{}
	default:
		return nil, fmt.Errorf("unknown event: %v", eventID)
	}

	return inst, json.Unmarshal(data, inst)
}
