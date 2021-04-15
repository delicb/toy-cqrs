package types

const UserCreatedEventID EventID = "user.created"

type UserCreatedParams struct {
	Email     string `json:"email,omitempty"`
	Password  string `json:"password,omitempty"`
	IsEnabled bool   `json:"is_enabled,omitempty"`
}

const UserEmailChangedEventID EventID = "user.email_changed"

type UserEmailChangedParams struct {
	OldEmail string `json:"old_email,omitempty"`
	NewEmail string `json:"new_email,omitempty"`
}

const UserPasswordChangedEventID EventID = "user.password_changed"

type UserPasswordChangedParams struct {
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

const UserEnabledEventID EventID = "user.enabled"

type UserEnabledParams struct{}

const UserDisabledEventID EventID = "user.disabled"

type UserDisabledParams struct{}
