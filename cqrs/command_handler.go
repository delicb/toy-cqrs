package cqrs

import (
	"errors"
	"log"

	"go.uber.org/multierr"
)

// CommandHandler can process and execute a command.
type CommandHandler interface {
	// HandleCommand processes provided command or dies (returns error) trying.
	HandleCommand(cmd Command) error
}

type CommandValidator interface {
	Validate(Command) error
}

type simpleCommandHandler struct {
	repo       Repository
	validators []CommandValidator
}

func (h *simpleCommandHandler) HandleCommand(cmd Command) error {
	// overview of an algorithm
	// - recreate user from previous events
	// - validate command
	// - generate new events and apply to user
	// - save to database
	// - publish new events

	// recreate user from past events
	root, err := h.repo.Load(cmd.GetAggregateType(), cmd.GetAggregateID())
	if err != nil {
		return err
	}
	log.Printf("have root: %+v\n", root)

	// validate command
	if err := cmd.Validate(root); err != nil {
		return err
	}

	// call 3rd party validators to allow them to report errors
	var validationError error
	for _, validator := range h.validators {
		validationError = multierr.Combine(validationError, validator.Validate(cmd))
	}
	if validationError != nil {
		return validationError
	}

	// generate and apply new events
	if err := root.HandleCommand(cmd); err != nil {
		return err
	}

	log.Println("handled command")

	// just a sanity check
	if root.GetID() == "" {
		return errors.New("aggregate root ID not populated and it should have been by this point")
	}

	log.Println("have root id:", root.GetID())

	// save, publish happens automatically with our postgres implementation
	return h.repo.Save(root)
}

// AddValidator add new implementation of CommandValidator to be called during command handling.
func (h *simpleCommandHandler) AddValidator(v CommandValidator) {
	h.validators = append(h.validators, v)
}

func NewSimpleHandler(repo Repository) *simpleCommandHandler {
	return &simpleCommandHandler{
		repo: repo,
	}
}
