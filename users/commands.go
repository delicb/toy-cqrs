package users

import (
	"errors"
	"strings"

	"github.com/delicb/toy-cqrs/cqrs"
)

var (
	CommandSerializer cqrs.CommandSerializer
)

func init() {
	serializer := cqrs.NewCommandJSONSerializer()
	serializer.RegisterCommandCtor(CreateUserID, func() cqrs.Command { return &CreateUser{} })
	serializer.RegisterCommandCtor(ChangeUserEmailID, func() cqrs.Command { return &ChangeUserEmail{} })
	serializer.RegisterCommandCtor(ChangeUserPasswordID, func() cqrs.Command { return &ChangeUserPassword{} })
	serializer.RegisterCommandCtor(EnableUserID, func() cqrs.Command { return &EnableUser{} })
	serializer.RegisterCommandCtor(DisableUserID, func() cqrs.Command { return &DisableUser{} })

	CommandSerializer = serializer
}

const CreateUserID cqrs.CommandID = "user.create"
const ChangeUserEmailID cqrs.CommandID = "user.change.email"
const ChangeUserPasswordID cqrs.CommandID = "user.change.password"
const EnableUserID cqrs.CommandID = "user.enable"
const DisableUserID cqrs.CommandID = "user.disable"

// CreateUser is command indicating that new user should be created.
type CreateUser struct {
	cqrs.BaseCommand `mapstructure:",squash"`
	Email            string `json:"email" mapstructure:"email"`
	Password         string `json:"password" mapstructure:"password"`
}

func (c *CreateUser) Validate(root cqrs.AggregateRoot) error {
	if root.GetID() != "" {
		return errors.New("user ID should not be set for create user command")
	}
	if !strings.HasPrefix(c.Password, "bcrypt") {
		return errors.New("password not hashed")
	}
	return nil
}

// ChangeUserEmail is command indicating that existing user's email should be changed.
type ChangeUserEmail struct {
	cqrs.BaseCommand `mapstructure:",squash"`
	Email            string `json:"email" mapstructure:"email"`
}

// ChangeUserPassword is command indicating that existing user's password should be changed.
type ChangeUserPassword struct {
	cqrs.BaseCommand `mapstructure:",squash"`
	Password         string `json:"password" mapstructure:"password"`
}

func (c *ChangeUserPassword) Validate(_ cqrs.AggregateRoot) error {
	if !strings.HasPrefix(c.Password, "bcrypt") {
		return errors.New("password not hashed")
	}
	return nil
}

// EnableUser is command indicating that existing user should be enabled.
type EnableUser struct {
	cqrs.BaseCommand `mapstructure:",squash"`
}

// DisableUser is command indicating that existing user should be disabled.
type DisableUser struct {
	cqrs.BaseCommand `mapstructure:",squash"`
}
