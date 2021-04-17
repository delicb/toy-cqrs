package main

import (
	"log"
	"time"

	"github.com/google/uuid"
)

// User is main domain entity for this service.
type User struct {
	Root

	Email     string
	Password  string
	IsEnabled bool
}

func (u *User) Apply(new bool, ev *Event) error {
	switch d := ev.Data.(type) {
	case *UserCreated:
		log.Println("applying user created command")
		u.ID = ev.AggregateID
		u.Email = d.Email
		u.Password = d.Password
		u.IsEnabled = d.IsEnabled
	case *UserEmailChanged:
		u.Email = d.NewEmail
	case *UserPasswordChanged:
		u.Password = d.NewPassword
	case *UserEnabled:
		u.IsEnabled = true
	case *UserDisabled:
		u.IsEnabled = false
	default:
		return ErrUnknownEvent(ev.Data)
	}

	// do this at the end, in order not to save unknown event
	if new {
		u.Changes = append(u.Changes, ev)
	}
	return nil
}

func (u *User) HandleCommand(cmd Command) error {
	log.Printf("handling command: %T\n", cmd)
	switch c := cmd.(type) {
	case *CreateUser:
		newUserID := uuid.NewString()
		ev := NewEv(UserCreatedID, cmd, &UserCreated{
			ID:        newUserID,
			Email:     c.Email,
			Password:  c.Password,
			IsEnabled: false,
		})
		ev.AggregateID = newUserID
		return u.Apply(true, ev)
	case *ChangeUserEmail:
		return u.Apply(true, NewEv(EmailChangedID, cmd, &UserEmailChanged{
			NewEmail: c.Email,
			OldEmail: u.Email,
		}))
	case *ChangeUserPassword:
		return u.Apply(true, NewEv(PasswordChangedID, cmd, &UserPasswordChanged{
			NewPassword: c.Password,
			OldPassword: u.Password,
		}))
	case *EnableUser:
		return u.Apply(true, NewEv(EnabledID, cmd, &UserEnabled{}))
	case *DisableUser:
		return u.Apply(true, NewEv(DisabledID, cmd, &UserDisabled{}))
	default:
		return ErrUnknownCommand(cmd)
	}
}

func NewEv(id EventID, cmd Command, data interface{}) *Event {
	return &Event{
		EventID:       id,
		AggregateID:   cmd.GetAggregateID(),
		CreatedAt:     time.Now().UTC(),
		CorrelationID: cmd.GetCorrelationID(),
		Data:          data,
	}
}
