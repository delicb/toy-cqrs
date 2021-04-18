package main

import (
	"fmt"

	"github.com/delicb/toy-cqrs/cqrs"
)

type unknownCommandError struct {
	cmd cqrs.Command
}

func (e *unknownCommandError) Error() string {
	return fmt.Sprintf("unknown command: %T", e.cmd)
}

func ErrUnknownCommand(cmd cqrs.Command) error {
	return &unknownCommandError{cmd}
}

type unknownEventError struct {
	event interface{}
}

func (e *unknownEventError) Error() string {
	return fmt.Sprintf("unknown event: %T", e.event)
}

func ErrUnknownEvent(evt interface{}) error {
	return &unknownEventError{evt}
}

type commandValidationError struct {
	cmd cqrs.Command
	msg string
}

func (e *commandValidationError) Error() string {
	return fmt.Sprintf("command validation error for command: %T: %v", e.cmd, e.msg)
}

func ErrCommandValidation(cmd cqrs.Command, msg string) error {
	return &commandValidationError{
		cmd: cmd,
		msg: msg,
	}
}
