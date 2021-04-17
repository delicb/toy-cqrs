package main

import (
	"strings"
)

// Command is something sent to a service to execute.
type Command interface {
	// Validate checks if this command is valid to be executed on provided aggregate root.
	// If not, error should be return, nil otherwise.
	Validate(root AggregateRoot) error

	// GetAggregateID returns unique ID of an object that this command should be applied on.
	// Only in case of create events this should return zero value and aggregate root
	// handler for such commands should expect that.
	GetAggregateID() string

	// GetAggregateType returns identification of a type to which this command should be
	// applied to. Together with GetAggregateID this should determine instance on which
	// command should be applied to.
	GetAggregateType() string

	// GetCorrelationID returns unique identifier associated with concrete command execution.
	// It is used to tie all events that were created from the same command, but also to identify
	// messages passed between systems that apply to the same command.
	GetCorrelationID() string
}

// BaseCommand is utility, implementing common parts for each command.
type BaseCommand struct {
	AggregateID   string `json:"aggregate_id,omitempty"`
	AggregateType string `json:"aggregate_type,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (c *BaseCommand) Validate(_ AggregateRoot) error { return nil }
func (c *BaseCommand) GetAggregateID() string         { return c.AggregateID }
func (c *BaseCommand) GetAggregateType() string       { return c.AggregateType }
func (c *BaseCommand) GetCorrelationID() string       { return c.CorrelationID }

// CreateUser is command indicating that new user should be created.
type CreateUser struct {
	BaseCommand
	Email    string
	Password string
}

func (c *CreateUser) Validate(root AggregateRoot) error {
	if root.GetID() != "" {
		return ErrCommandValidation(c, "user ID should not be set")
	}
	if !strings.HasPrefix(c.Password, "bcrypt") {
		return ErrCommandValidation(c, "password not hashed")
	}
	return nil
}

// ChangeUserEmail is command indicating that existing user's email should be changed.
type ChangeUserEmail struct {
	BaseCommand
	Email string
}

// ChangeUserPassword is command indicating that existing user's password should be changed.
type ChangeUserPassword struct {
	BaseCommand
	Password string
}

func (c *ChangeUserPassword) Validate(_ AggregateRoot) error {
	if !strings.HasPrefix(c.Password, "bcrypt") {
		return ErrCommandValidation(c, "password not hashed")
	}
	return nil
}

// EnableUser is command indicating that existing user should be enabled.
type EnableUser struct {
	BaseCommand
}

// DisableUser is command indicating that existing user should be disabled.
type DisableUser struct {
	BaseCommand
}
