package main

import (
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"

	"github.com/delicb/toy-cqrs/types"
)

// UserCommandHandler describes what single command handler should be able to do.
// It is specific to main domain entity for this service.
type UserCommandHandler interface {
	RecreateUser(types.Command) (*User, error)
	Validate(*User, types.Command) error
	Events(*User, types.Command) ([]types.Event, error)
}

type createCommandHandler struct{}

func (c *createCommandHandler) RecreateUser(_ types.Command) (*User, error) {
	// for create user command, there should be no user to recreate, since we are seeing the user
	// for the first time, so just return empty user
	return &User{}, nil
}

func (c *createCommandHandler) Validate(_ *User, cmd types.Command) error {
	payload := &types.CreateUserCmdParams{}
	if err := cmd.LoadPayload(payload); err != nil {
		return err
	}

	// verify if password is encrypted, we do not want to continue with plaintext password
	if !strings.HasPrefix(payload.Password, "bcrypt") {
		return errors.New("unencrypted password")
	}

	// TODO: figure out how to check for email uniqueness
	return nil
}

func (c *createCommandHandler) Events(_ *User, cmd types.Command) ([]types.Event, error) {
	payload := &types.CreateUserCmdParams{}
	if err := cmd.LoadPayload(payload); err != nil {
		return nil, err
	}
	events := make([]types.Event, 0)
	newUserID := uuid.NewString()
	log.Println("--> Creating user: ", newUserID)
	ev, err := types.NewEvent(types.UserCreatedEventID, newUserID, cmd.CorrelationID(), &types.UserCreatedParams{
		Email:     payload.Email,
		Password:  payload.Password,
		IsEnabled: false,
	})
	if err != nil {
		return nil, err
	}
	return append(events, ev), nil
}

// dummy interface for getting events, db manager implements it
type eventGetter interface {
	GetEvents(id string) ([]types.Event, error)
}
type modifyUserHandler struct {
	evs eventGetter
}

func (c *modifyUserHandler) RecreateUser(cmd types.Command) (*User, error) {
	payload := &types.ModifyUserCmdParams{}
	if err := cmd.LoadPayload(payload); err != nil {
		return nil, err
	}

	log.Println("getting events for aggregate ID:", payload.ID)
	oldEvents, err := c.evs.GetEvents(payload.ID)
	if err != nil {
		return nil, err
	}
	u := &User{}
	return u, u.Apply(oldEvents...)
}

func (c *modifyUserHandler) Validate(_ *User, cmd types.Command) error {
	// TODO: implement validation
	payload := &types.ModifyUserCmdParams{}
	if err := cmd.LoadPayload(payload); err != nil {
		return err
	}
	if payload.ID == "" {
		return errors.New("user ID not provided")
	}

	return nil
}

func (c *modifyUserHandler) Events(u *User, cmd types.Command) ([]types.Event, error) {
	payload := &types.ModifyUserCmdParams{}
	if err := cmd.LoadPayload(payload); err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("user not provided")
	}
	events := make([]types.Event, 0)

	log.Printf("creating modify events for user: %+v", u)

	if payload.Email != nil && u.Email != *payload.Email {
		ev, err := types.NewEvent(types.UserEmailChangedEventID, payload.ID, cmd.CorrelationID(), &types.UserEmailChangedParams{
			OldEmail: u.Email,
			NewEmail: *payload.Email,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}

	if payload.Password != nil && u.Password != *payload.Password {
		ev, err := types.NewEvent(types.UserPasswordChangedEventID, payload.ID, cmd.CorrelationID(), &types.UserPasswordChangedParams{
			OldPassword: u.Password,
			NewPassword: *payload.Password,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}

	if payload.IsEnabled != nil && u.IsEnabled != *payload.IsEnabled {
		evtID := types.UserDisabledEventID
		if *payload.IsEnabled {
			evtID = types.UserEnabledEventID
		}
		ev, err := types.NewEvent(evtID, payload.ID, cmd.CorrelationID(), nil)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil

}

type allCommandHandler struct {
	real map[types.CommandID]UserCommandHandler
}

func (c *allCommandHandler) RecreateUser(cmd types.Command) (*User, error) {
	if h, ok := c.real[cmd.ID()]; ok {
		return h.RecreateUser(cmd)
	}
	return nil, UnknownCommandErr(cmd.ID())
}

func (c *allCommandHandler) Validate(user *User, cmd types.Command) error {
	if h, ok := c.real[cmd.ID()]; ok {
		return h.Validate(user, cmd)
	}
	return UnknownCommandErr(cmd.ID())
}

func (c *allCommandHandler) Events(user *User, cmd types.Command) ([]types.Event, error) {
	if h, ok := c.real[cmd.ID()]; ok {
		return h.Events(user, cmd)
	}
	return nil, UnknownCommandErr(cmd.ID())
}

func NewUserCommandHandler(evs eventGetter) UserCommandHandler {
	return &allCommandHandler{real: map[types.CommandID]UserCommandHandler{
		types.CreateUserCmdID: &createCommandHandler{},
		types.ModifyUserCmdID: &modifyUserHandler{evs},
	}}
}
