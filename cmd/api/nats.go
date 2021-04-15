package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/delicb/toy-cqrs/types"
)

// NatsManager provides operations for managing events on nats server
type NatsManager interface {
	// Subscribe creates subscription for provided correlation ID. It returns a channel
	// on which a client can wait on. This channel is also stored internally and read
	// if client calls WaitForEvent. If client is reading from the channel, it should
	// not call WaitForEvent and vice versa.
	// It is save to call Subscribe and a bit later call WaitForEvent, messages
	// received in meantime will be saved (buffer is 1 message, but that should be
	// enough, given that only one message is expected per correlationID).
	// Second returned channel is error channel, on which message will be sent in
	// case error message is received from nats.
	Subscribe(correlationID string) (<-chan string, <-chan error, error)

	// Unsubscribe removes subscription for provided correlation ID.
	Unsubscribe(correlationID string) error

	// WaitForEvent blocks until event for provided correlation ID is received.
	// If message is received before the call, but after the call to Subscribe,
	// method should return immediately, otherwise it waits for the message or
	// provided timeout.
	WaitForEvent(correlationID string, timeout time.Duration) (string, error)

	// SendCommand sends command to appropriate destination.
	SendCommand(cmd types.Command) error
}

type activeSubscription struct {
	ch    <-chan string
	errCh <-chan error
	sub   *nats.Subscription
}

type natsManager struct {
	conn          *nats.Conn
	subscriptions map[string]*activeSubscription
}

// NewNatsManager returns instance of a nats manager specific for use cases
// of this service.
func NewNatsManager(url string) *natsManager {
	conn, err := nats.Connect(url)
	if err != nil {
		panic(err)
	}
	return &natsManager{
		conn:          conn,
		subscriptions: map[string]*activeSubscription{},
	}
}

func (n *natsManager) Subscribe(correlationID string) (<-chan string, <-chan error, error) {
	subject := fmt.Sprintf("event.%v.*", correlationID)
	ch := make(chan string, 1)
	errCh := make(chan error, 1)
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if strings.HasSuffix(msg.Subject, "error") {
			errCh <- errors.New(string(msg.Data))
		} else {
			ch <- string(msg.Data)
		}
	})
	if err == nil {
		n.subscriptions[correlationID] = &activeSubscription{
			ch:    ch,
			errCh: errCh,
			sub:   sub,
		}
	}
	return ch, errCh, err
}

func (n *natsManager) Unsubscribe(correlationID string) error {
	if sub, ok := n.subscriptions[correlationID]; ok {
		return sub.sub.Unsubscribe()
	}
	return nil
}

func (n *natsManager) WaitForEvent(correlationID string, timeout time.Duration) (string, error) {
	sub, ok := n.subscriptions[correlationID]
	if !ok {
		return "", fmt.Errorf("unknown correlation ID: %v (call to Subscribe needed first)", correlationID)
	}
	select {
	case d := <-sub.ch:
		return d, nil
	case err := <-sub.errCh:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout")
	}
}

func (n *natsManager) SendCommand(cmd types.Command) error {
	sub := fmt.Sprintf("command.%v", cmd.ID())
	payload, err := cmd.Marshal()
	if err != nil {
		return err
	}
	resp, err := n.conn.Request(sub, payload, 1*time.Second)
	if err != nil {
		return err
	}
	r := string(resp.Data)
	if strings.HasPrefix(r, "error:") {
		return errors.New(strings.TrimPrefix(r, "error:"))
	}
	return nil
}
