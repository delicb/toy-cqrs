package main

import (
	"log"

	"github.com/google/uuid"

	"github.com/delicb/toy-cqrs/cqrs"
	"github.com/delicb/toy-cqrs/users"
)

// User is main domain entity for this service.
type User struct {
	cqrs.Root

	Email     string
	Password  string
	IsEnabled bool
}

func (u *User) Apply(new bool, ev *cqrs.Event) error {
	switch d := ev.Data.(type) {
	case *users.UserCreated:
		log.Println("applying user created command")
		u.ID = ev.AggregateID
		u.Email = d.Email
		u.Password = d.Password
		u.IsEnabled = d.IsEnabled
	case *users.UserEmailChanged:
		u.Email = d.NewEmail
	case *users.UserPasswordChanged:
		u.Password = d.NewPassword
	case *users.UserEnabled:
		u.IsEnabled = true
	case *users.UserDisabled:
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

func (u *User) HandleCommand(cmd cqrs.Command) error {
	log.Printf("handling command: %T\n", cmd)
	switch c := cmd.(type) {
	case *users.CreateUser:
		newUserID := uuid.NewString()
		ev := cqrs.NewEvent(users.UserCreatedID, cmd, &users.UserCreated{
			ID:        newUserID,
			Email:     c.Email,
			Password:  c.Password,
			IsEnabled: false,
		})
		ev.AggregateID = newUserID
		return u.Apply(true, ev)
	case *users.ChangeUserEmail:
		return u.Apply(true, cqrs.NewEvent(users.EmailChangedID, cmd, &users.UserEmailChanged{
			NewEmail: c.Email,
			OldEmail: u.Email,
		}))
	case *users.ChangeUserPassword:
		return u.Apply(true, cqrs.NewEvent(users.PasswordChangedID, cmd, &users.UserPasswordChanged{
			NewPassword: c.Password,
			OldPassword: u.Password,
		}))
	case *users.EnableUser:
		return u.Apply(true, cqrs.NewEvent(users.EnabledID, cmd, &users.UserEnabled{}))
	case *users.DisableUser:
		return u.Apply(true, cqrs.NewEvent(users.DisabledID, cmd, &users.UserDisabled{}))
	default:
		return ErrUnknownCommand(cmd)
	}
}
