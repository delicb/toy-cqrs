package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"go.uber.org/multierr"
)

type EventStorage interface {
	Load(id string) ([]*Event, error)
	Save([]*Event) error
}

type inMemoryStorage struct {
	state []*Event
}

func (m *inMemoryStorage) Load(id string) ([]*Event, error) {
	forID := make([]*Event, 0)
	for _, ev := range m.state {
		if ev.AggregateID == id {
			forID = append(forID, ev)
		}
	}
	return forID, nil
}
func (m *inMemoryStorage) Save(events []*Event) error {
	m.state = append(m.state, events...)
	return nil
}

type Repository interface {
	Load(id string) (AggregateRoot, error)
	Save(root AggregateRoot) error
}

type simpleRepository struct {
	storage EventStorage
}

func (r *simpleRepository) Load(id string) (AggregateRoot, error) {
	oldEvents, err := r.storage.Load(id)
	if err != nil {
		return nil, err
	}
	// TODO: create somehow, e.g. register constructor
	u := &User{}
	for _, ev := range oldEvents {
		if err := u.Apply(false, ev); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func (r *simpleRepository) Save(agg AggregateRoot) error {
	if err := r.storage.Save(agg.GetChanges()); err != nil {
		return err
	}
	agg.ClearChanges()
	return nil
}

type CommandHandler interface {
	HandleCommand(Command) error
}

type simpleCommandHandler struct {
	repo Repository
}

func (h *simpleCommandHandler) HandleCommand(cmd Command) error {
	// Algorithm
	// - recreate user from previous events
	// - validate command
	// - generate new events
	// - apply events to user
	// - save to database
	// - publish

	// recreate user from past events
	agg, err := h.repo.Load(cmd.GetAggregateID())
	if err != nil {
		return err
	}
	log.Printf("got recreated user: %+v", agg)

	if err := cmd.Validate(agg); err != nil {
		return err
	}

	if err := agg.HandleCommand(cmd); err != nil {
		return err
	}

	// at this point aggregate ID has to be populated, if not something went very wrong
	if agg.GetID() == "" {
		return errors.New("aggregate root ID not populated")
	}

	log.Printf("%+v\n", agg)

	return h.repo.Save(agg)
}

type AggregateRoot interface {
	HandleCommand(cmd Command) error
	GetID() string
	GetChanges() []*Event
	ClearChanges()
	Apply(new bool, ev *Event) error
}

type Root struct {
	ID      string
	Changes []*Event
}

func (r *Root) GetID() string        { return r.ID }
func (r *Root) GetChanges() []*Event { return r.Changes }
func (r *Root) ClearChanges()        { r.Changes = []*Event{} }

type User struct {
	Root

	Email     string
	Password  string
	IsEnabled bool
}

func (u *User) Apply(new bool, ev *Event) error {

	switch d := ev.Data.(type) {
	case *UserCreated:
		u.ID = d.ID
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
		return fmt.Errorf("unkonwn event: %T", ev.Data)
	}

	// do this at the end, in order not to save unknown event
	if new {
		u.Changes = append(u.Changes, ev)
	}
	return nil
}

func (u *User) HandleCommand(cmd Command) error {

	switch c := cmd.(type) {
	case *CreateUser:
		newUserID := uuid.NewString()
		return u.Apply(true, &Event{"", newUserID, &UserCreated{
			ID:        newUserID,
			Password:  c.Password,
			Email:     c.Email,
			IsEnabled: false,
		}})
	case *ChangeUserEmail:
		return u.Apply(true, &Event{"", u.GetID(), &UserEmailChanged{
			NewEmail: u.Email,
			OldEmail: c.Email,
		}})
	case *ChangeUserPassword:
		return u.Apply(true, &Event{"", u.GetID(), &UserPasswordChanged{
			NewPassword: c.Password,
			OldPassword: u.Password,
		}})
	case *EnableUser:
		return u.Apply(true, &Event{"", u.GetID(), &UserEnabled{}})
	case *DisableUser:
		return u.Apply(true, &Event{"", u.GetID(), &UserDisabled{}})
	default:
		return fmt.Errorf("unknown command: %T", cmd)
	}
}

type Event struct {
	ID          string
	AggregateID string
	Data        interface{}
}

type UserCreated struct {
	ID        string
	Password  string
	Email     string
	IsEnabled bool
}

type UserEmailChanged struct {
	NewEmail string
	OldEmail string
}

type UserPasswordChanged struct {
	NewPassword string
	OldPassword string
}

type UserEnabled struct{}
type UserDisabled struct{}

type Command interface {
	Validate(root AggregateRoot) error
	GetAggregateID() string
	GetAggregateType() string
}

type BaseCommand struct {
	AggregateID   string
	AggregateType string
}

func (c *BaseCommand) Validate(root AggregateRoot) error { return nil }
func (c *BaseCommand) GetAggregateID() string            { return c.AggregateID }
func (c *BaseCommand) GetAggregateType() string          { return c.AggregateType }

type CreateUser struct {
	BaseCommand
	Email    string
	Password string
}

func (c *CreateUser) Validate(root AggregateRoot) error {
	u, ok := root.(*User)
	if !ok {
		return ErrValidation("user", "command not applicable")
	}

	if u.ID != "" || u.Email != "" || u.Password != "" {
		return ErrValidation("user", "already populated")
	}
	return nil
}

type ChangeUserEmail struct {
	BaseCommand
	ID    string
	Email string
}

type ChangeUserPassword struct {
	BaseCommand
	ID       string
	Password string
}

type EnableUser struct {
	BaseCommand
}

type DisableUser struct {
	BaseCommand
}

type validationError struct {
	entity  string
	message string
}

func (e *validationError) Error() string {
	return fmt.Sprintf("validation error on type: %v: %v", e.entity, e.message)
}

func ErrValidation(entity string, msg string) error { return &validationError{entity, msg} }

func main() {
	storage := &inMemoryStorage{state: make([]*Event, 0)}
	repo := &simpleRepository{storage}
	handler := &simpleCommandHandler{repo}

	err := multierr.Combine(
		handler.HandleCommand(&CreateUser{
			// BaseCommand: BaseCommand{},
			Email:    "bojan@delic.in.rs",
			Password: "testpass",
		}),
		handler.HandleCommand(&EnableUser{
			BaseCommand: BaseCommand{AggregateID: "unkonwn"},
		}),
	)
	if err != nil {
		panic(err)
	}
}
