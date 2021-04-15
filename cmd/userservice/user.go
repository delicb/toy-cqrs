package main

import (
	"log"

	"github.com/delicb/toy-cqrs/types"
)

// User is main domain entity for this service.
type User struct {
	ID        string
	Email     string
	Password  string
	IsEnabled bool
}

func (u *User) Apply(events ...types.Event) error {
	log.Printf("applying %d events...", len(events))
	for _, ev := range events {
		log.Println("applying event to user: ", ev.ID())
		switch ev.ID() {
		case types.UserCreatedEventID:
			payload := &types.UserCreatedParams{}
			if err := ev.LoadPayload(payload); err != nil {
				return err
			}
			u.ID = ev.AggregateID()
			u.Email = payload.Email
			u.Password = payload.Password
			u.IsEnabled = false
		case types.UserEmailChangedEventID:
			payload := &types.UserEmailChangedParams{}
			if err := ev.LoadPayload(payload); err != nil {
				return err
			}
			u.Email = payload.NewEmail
		case types.UserEnabledEventID:
			u.IsEnabled = true
		case types.UserDisabledEventID:
			u.IsEnabled = false
		default:
			return UnknownEventErr(ev.ID())
		}
	}
	return nil
}
