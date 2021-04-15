package types

const CreateUserCmdID CommandID = "user.create"
const ModifyUserCmdID CommandID = "user.modify"

type CreateUserCmdParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewCreateUser creates command for user creation.
func NewCreateUser(c CreateUserCmdParams) (Command, error) { return NewCommand(CreateUserCmdID, c) }

type ModifyUserCmdParams struct {
	ID        string  `json:"id"`
	Email     *string `json:"email,omitempty"`
	Password  *string `json:"password,omitempty"`
	IsEnabled *bool   `json:"is_enabled,omitempty"`
}

// NewModifyUser creates command for modifying user.
func NewModifyUser(c ModifyUserCmdParams) (Command, error) { return NewCommand(ModifyUserCmdID, c) }
