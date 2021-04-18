package main

import (
	"fmt"

	"github.com/delicb/toy-cqrs/cqrs"
	"github.com/delicb/toy-cqrs/users"
)

type validator struct {
	db         *psqlEventStorage
	emailState map[string]struct{}
}

func NewValidator(db *psqlEventStorage) (*validator, error) {
	v := &validator{
		db:         db,
		emailState: make(map[string]struct{}),
	}
	return v, v.init()
}

func (v *validator) Validate(cmd cqrs.Command) error {
	switch c := cmd.(type) {
	case *users.CreateUser:
		if _, ok := v.emailState[c.Email]; ok {
			return fmt.Errorf("email %q taken", c.Email)
		}
	case *users.ChangeUserEmail:
		if _, ok := v.emailState[c.Email]; ok {
			return fmt.Errorf("email %q taken", c.Email)
		}
	}
	return nil
}

func (v *validator) init() error {
	emailEvents, err := v.db.LoadEmailEvents()
	if err != nil {
		return err
	}
	for _, ev := range emailEvents {
		v.UpdateEmailState(ev)
	}
	return nil
}

func (v *validator) UpdateEmailState(ev *cqrs.Event) {
	switch d := ev.Data.(type) {
	case *users.UserCreated:
		v.emailState[d.Email] = struct{}{}
	case *users.UserEmailChanged:
		delete(v.emailState, d.OldEmail)
		v.emailState[d.NewEmail] = struct{}{}
	}
}
