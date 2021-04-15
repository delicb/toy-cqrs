package main

import (
	"fmt"

	"github.com/delicb/toy-cqrs/types"
)

type unknownCommandError struct {
	CommandID types.CommandID
}

func (uce *unknownCommandError) Error() string {
	return fmt.Sprintf("unknown command: %v", uce.CommandID)
}

func UnknownCommandErr(cmd types.CommandID) error {
	return &unknownCommandError{cmd}
}

type unknownEventError struct {
	EventID types.EventID
}

func (uee *unknownEventError) Error() string {
	return fmt.Sprintf("unknown event: %v", uee.EventID)
}

func UnknownEventErr(evt types.EventID) error {
	return &unknownEventError{evt}
}
