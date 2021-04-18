package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/multierr"

	"github.com/delicb/toy-cqrs/cqrs"
)

// Client describes commands that can be executed on user service.
type Client interface {
	Create(email, password string) (userID string, err error)
	ChangeEmail(userID, email string) error
	ChangePassword(userID, password string) error
	Enable(userID string) error
	Disable(userID string) error
}

type userClient struct {
	nm *nm
}

// NewClient returns instance of a user client.
func NewClient(conn *nats.Conn) *userClient {
	return &userClient{
		nm: &nm{
			conn:          conn,
			subscriptions: make(map[string]*activeSub),
		},
	}
}

func (c *userClient) Create(email, password string) (userID string, err error) {
	log.Println("creating user")
	correlationID := uuid.NewString()

	cmd := &CreateUser{
		BaseCommand: cqrs.BaseCommand{
			CommandID:     CreateUserID,
			AggregateID:   "",
			AggregateType: "user",
			CorrelationID: correlationID,
		},
		Email:    email,
		Password: password,
	}

	log.Println("sending command.user.create")
	resp, err := c.SendCommandAndWait("create", correlationID, cmd, 5*time.Second)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (c *userClient) ChangeEmail(userID, email string) error {
	correlationID := uuid.NewString()
	cmd := &ChangeUserEmail{
		BaseCommand: cqrs.BaseCommand{
			CommandID:     ChangeUserEmailID,
			AggregateID:   userID,
			AggregateType: "user",
			CorrelationID: correlationID,
		},
		Email: email,
	}

	_, err := c.SendCommandAndWait("change.email", correlationID, cmd, 5*time.Second)
	return err
}

func (c *userClient) ChangePassword(userID, password string) error {
	correlationID := uuid.NewString()
	cmd := &ChangeUserPassword{
		BaseCommand: cqrs.BaseCommand{
			CommandID:     ChangeUserPasswordID,
			AggregateID:   userID,
			AggregateType: "user",
			CorrelationID: correlationID,
		},
		Password: password,
	}

	_, err := c.SendCommandAndWait("change.password", correlationID, cmd, 5*time.Second)
	return err
}

func (c *userClient) Enable(userID string) error {
	correlationID := uuid.NewString()
	cmd := &EnableUser{BaseCommand: cqrs.BaseCommand{
		CommandID:     EnableUserID,
		AggregateID:   userID,
		AggregateType: "user",
		CorrelationID: correlationID,
	}}
	_, err := c.SendCommandAndWait("enable", correlationID, cmd, 5*time.Second)
	return err
}

func (c *userClient) Disable(userID string) error {
	correlationID := uuid.NewString()
	cmd := &DisableUser{BaseCommand: cqrs.BaseCommand{
		CommandID:     DisableUserID,
		AggregateID:   userID,
		AggregateType: "user",
		CorrelationID: correlationID,
	}}
	_, err := c.SendCommandAndWait("disable", correlationID, cmd, 5*time.Second)
	return err
}

func (c *userClient) SendCommandAndWait(cmdName, correlationID string, cmd interface{}, timeout time.Duration) (payload []byte, err error) {
	commandSubject := fmt.Sprintf("command.user.%s", cmdName)
	responseEventSubject := fmt.Sprintf("event.%v.*", correlationID)

	// subscribe to feedback before we send a command
	_, _, err = c.nm.Subscribe(responseEventSubject)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = multierr.Combine(err, c.nm.Unsubscribe(responseEventSubject))
	}()

	// send command
	if err := c.nm.SendCommand(commandSubject, cmd); err != nil {
		return nil, err
	}

	// block until we get a response
	return c.nm.WaitForEvent(responseEventSubject, timeout)
}

var _ Client = &userClient{}

type activeSub struct {
	ch    <-chan []byte
	errCh <-chan error
	sub   *nats.Subscription
}

// small nats manager, rename later
type nm struct {
	conn          *nats.Conn
	subscriptions map[string]*activeSub
}

func (n *nm) SendCommand(name string, cmd interface{}) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	r, err := n.conn.Request(name, data, 1*time.Second)
	if err != nil {
		return err
	}
	response := string(r.Data)

	// protocol is tha each response from request to user service starts with either "ok:" or "error:"
	if strings.HasPrefix(response, "error:") {
		return fmt.Errorf("error from user service: %v", strings.TrimPrefix(response, "error:"))
	}

	return nil
}

func (n *nm) Subscribe(subject string) (<-chan []byte, <-chan error, error) {
	ch := make(chan []byte, 1)
	errCh := make(chan error, 1)
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if strings.HasSuffix(msg.Subject, "error") {
			errCh <- errors.New(string(msg.Data))
		} else {
			ch <- msg.Data
		}
	})
	if err != nil {
		return nil, nil, err
	}
	n.subscriptions[subject] = &activeSub{
		ch:    ch,
		errCh: errCh,
		sub:   sub,
	}

	return ch, errCh, nil
}

func (n *nm) Unsubscribe(subject string) error {
	if sub, ok := n.subscriptions[subject]; ok {
		return sub.sub.Unsubscribe()
	}
	return nil
}

func (n *nm) WaitForEvent(subject string, timeout time.Duration) ([]byte, error) {
	activeSub, ok := n.subscriptions[subject]
	if !ok {
		return nil, fmt.Errorf("not subscribe to %v", subject)
	}

	select {
	case msg := <-activeSub.ch:
		return msg, nil
	case err := <-activeSub.errCh:
		return nil, err
	case <-time.After(timeout):
		return nil, errors.New("timeout error")
	}
}
