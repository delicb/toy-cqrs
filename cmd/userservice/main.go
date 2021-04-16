package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/delicb/toy-cqrs/types"
)

func main() {
	db, err := newDBManager(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	commandHandler := NewUserCommandHandler(db)

	natsConn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}

	sub, err := natsConn.Subscribe("command.user.*", messageHandler(commandHandler, db, natsConn))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			log.Println("ERROR: failed to unsubscribe from nats")
		}
	}()

	// wait for stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	sig := <-signalCh
	if err := sub.Drain(); err != nil {
		log.Printf("ERROR: nats drain failed: %v\n", err)
	}
	natsConn.Close()
	log.Println("got exist signal:", sig)
}

func messageHandler(commandHandler UserCommandHandler, persistence EventPersistenceStore, natsConn *nats.Conn) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		cmd, err := types.UnmarshalCommand(msg.Data)
		if err != nil {
			respondError(msg, err)
		}
		respondOk(msg)

		// recreate user from previous events
		user, err := commandHandler.RecreateUser(cmd)
		if err != nil {
			publishError(natsConn, cmd.CorrelationID(), err)
			return
		}

		// validate command, on it own and against current user state
		if err := commandHandler.Validate(user, cmd); err != nil {
			publishError(natsConn, cmd.CorrelationID(), err)
			return
		}

		// generate new events for this command
		newEvents, err := commandHandler.Events(user, cmd)
		if err != nil {
			publishError(natsConn, cmd.CorrelationID(), err)
			return
		}

		// apply new events to check if there are errors
		if err := user.Apply(newEvents...); err != nil {
			publishError(natsConn, cmd.CorrelationID(), err)
			return
		}

		// finally, save events to database
		if err := persistence.SaveEvents(newEvents); err != nil {
			publishError(natsConn, cmd.CorrelationID(), err)
			return
		}

		// if we got this far, we saved events to database, no need to
		// do anything else
	}
}

func respondError(msg *nats.Msg, err error) {
	if nerr := msg.Respond([]byte(fmt.Sprintf("error:%v", err))); nerr != nil {
		log.Printf("ERROR: failed to respond to nats message: %v\n", err)
	}
}

func respondOk(msg *nats.Msg) {
	if err := msg.Respond([]byte("ok:ack")); err != nil {
		log.Printf("ERROR: failed to respond to nats message: %v\n", err)
	}
}

func publishError(natsConn *nats.Conn, correlationID string, err error) {
	sub := fmt.Sprintf("event.%v.error", correlationID)
	if nerr := natsConn.Publish(sub, []byte(err.Error())); nerr != nil {
		log.Printf("ERROR: failed to publish nats error message with subject %v and error message: %v\n", sub, err)
	}
}
