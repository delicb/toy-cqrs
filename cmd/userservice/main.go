package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/delicb/toy-cqrs/cqrs"
	"github.com/delicb/toy-cqrs/users"
)

func main() {
	// root context
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// nats connection, we receive commands over nats
	natsConn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}

	// initialize storage
	store, err := NewPsqlEventStore(rootCtx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	// create validator to register with command handler
	validator, err := NewValidator(store)
	if err != nil {
		panic(err)
	}

	// register hook to update validator state when events are saved
	store.AddAfterSaveHook(validator.UpdateEmailState)

	// initialize aggregate root repository
	repo := cqrs.NewSimpleRepository(store)

	// register constructor for our main (and only) aggregate root (user)
	repo.RegisterCtor("user", func() cqrs.AggregateRoot { return &User{} })

	// create simple command handler
	handler := cqrs.NewSimpleHandler(repo)
	// hook validator into command handler
	handler.AddValidator(validator)

	log.Println("subscribing to commands")
	sub, err := natsConn.Subscribe("command.user.>", func(msg *nats.Msg) {
		// get command name from subject
		cmd, err := users.CommandSerializer.Unmarshal(msg.Data)
		if err != nil {
			respondError(msg, err)
			return
		}

		log.Printf("Have command: %+v", cmd)
		// respond that command is accepted
		respondOk(msg)

		if err := handler.HandleCommand(cmd); err != nil {
			log.Println("handing command failed with error: ", err)
			publishError(natsConn, cmd.GetCorrelationID(), err)
		}
	})
	if err != nil {
		panic(err)
	}

	log.Println("waiting for the stop signal")
	// wait for the stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	sig := <-signalCh
	log.Printf("got signal: %v, stopping", sig)
	if err := sub.Drain(); err != nil {
		log.Printf("ERROR: nats drain failed: %v\n", err)
	}
	natsConn.Close()
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
