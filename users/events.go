package users

import (
	"github.com/delicb/toy-cqrs/cqrs"
)

var (
	EventSerializer cqrs.EventSerializer
)

func init() {
	serializer := cqrs.NewEventJSONSerializer()
	serializer.RegisterDataCtor(UserCreatedID, func() interface{} { return &UserCreated{} })
	serializer.RegisterDataCtor(EmailChangedID, func() interface{} { return &UserEmailChanged{} })
	serializer.RegisterDataCtor(PasswordChangedID, func() interface{} { return &UserEmailChanged{} })
	serializer.RegisterDataCtor(EnabledID, func() interface{} { return &UserEnabled{} })
	serializer.RegisterDataCtor(DisabledID, func() interface{} { return &UserDisabled{} })

	EventSerializer = serializer
}

const UserCreatedID cqrs.EventID = "user.created"
const EmailChangedID cqrs.EventID = "user.email.changed"
const PasswordChangedID cqrs.EventID = "user.password.changed"
const EnabledID cqrs.EventID = "user.enabled"
const DisabledID cqrs.EventID = "user.disabled"

// UserCreated is event indicating that new user has been created.
type UserCreated struct {
	ID        string `json:"id,omitempty" mapstructure:"id"`
	Email     string `json:"email,omitempty" mapstructure:"email"`
	Password  string `json:"password,omitempty" mapstructure:"password"`
	IsEnabled bool   `json:"is_enabled,omitempty" mapstructure:"is_enabled"`
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
