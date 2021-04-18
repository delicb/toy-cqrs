package cqrs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// CommandID is identifier of a command across the system.
// Good idea is to use prefix with entity name.
type CommandID string

// Command is something sent to a service to execute.
type Command interface {
	// GetCommandID returns string uniquely identifying command in the system.
	GetCommandID() CommandID

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
// Intended usage is by embedding it to concrete command. E.g.
//   type MyCommand {
//     cqrs.BaseCommand
//     Param string
//    }
type BaseCommand struct {
	CommandID     CommandID `json:"command_id" mapstructure:"command_id"`
	AggregateID   string    `json:"aggregate_id" mapstructure:"aggregate_id"`
	AggregateType string    `json:"aggregate_type" mapstructure:"aggregate_type"`
	CorrelationID string    `json:"correlation_id" mapstructure:"correlation_id"`
}

func (c *BaseCommand) GetCommandID() CommandID        { return c.CommandID }
func (c *BaseCommand) Validate(_ AggregateRoot) error { return nil }
func (c *BaseCommand) GetAggregateID() string         { return c.AggregateID }
func (c *BaseCommand) GetAggregateType() string       { return c.AggregateType }
func (c *BaseCommand) GetCorrelationID() string       { return c.CorrelationID }

// CommandSerializer defines operations needed for command instance marshal and unmarshal operations.
type CommandSerializer interface {
	Marshal(Command) ([]byte, error)
	Unmarshal([]byte) (Command, error)
}

type commandJSONSerializer struct {
	ctors map[CommandID]func() Command
}

// NewCommandJSONSerializer is implementation of CommandSerializer that uses JSON as underlying format.
func NewCommandJSONSerializer() *commandJSONSerializer {
	return &commandJSONSerializer{
		ctors: make(map[CommandID]func() Command),
	}
}

func (s *commandJSONSerializer) RegisterCommandCtor(ID CommandID, ctor func() Command) {
	s.ctors[ID] = ctor
}

func (s *commandJSONSerializer) Marshal(cmd Command) ([]byte, error) {
	return json.Marshal(cmd)
}

func (s *commandJSONSerializer) Unmarshal(rawData []byte) (Command, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal(rawData, &raw); err != nil {
		return nil, err
	}
	cmdID, ok := raw["command_id"]
	if !ok {
		return nil, errors.New("raw data does not contain command_id")
	}
	cmdIdAsStr, ok := cmdID.(string)
	if !ok {
		return nil, fmt.Errorf("command_id has unexpected type: %T", cmdID)
	}

	ctor, ok := s.ctors[CommandID(cmdIdAsStr)]
	if !ok {
		return nil, fmt.Errorf("unknown command ID: %s", cmdIdAsStr)
	}
	cmd := ctor()

	return cmd, mapstructure.Decode(raw, cmd)
}
