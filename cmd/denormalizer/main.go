package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/nats-io/nats.go"
)

const notificationChannel = "new_event"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting denormalizer")
	// connect to database to receive events
	pool, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	natsConn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}

	usersDbManager := &dbManager{pool}

	publishManager := &natsManager{natsConn}

	events := make(chan *Event, 32)

	// start listener
	go listen(pool, events)
	// start event processor
	go eventProcessor(usersDbManager, publishManager, events)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	// wait for termination signal
	sig := <-signalCh
	fmt.Printf("got signal: %v, terminating", sig)

	// cleanup
	// pool.Close() // TODO: Check why this blocks when shutting down
	close(events)

}

func listen(pool *pgxpool.Pool, events chan<- *Event) {

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		log.Println("--> error getting connection from pool: ", err)

	}
	defer conn.Release()

	_, err = conn.Exec(context.Background(), "listen "+notificationChannel)
	if err != nil {
		log.Println("--> error listening for events:", err)
	}

	for {
		// blocks
		notification, err := conn.Conn().WaitForNotification(context.Background())

		if err != nil {
			log.Println("--> error getting notification from database")
		}

		if notification.Channel != notificationChannel {
			log.Println("got message from unexpected channel:", notification.Channel)
			continue
		}

		// ev, err := types.UnmarshalEvent([]byte(notification.Payload))
		ev, err := UnmarshalEvent([]byte(notification.Payload))
		if err != nil {
			log.Println("failed to unmarshal event from database into event structure:", err)
			continue
		}

		events <- ev
	}
}

func eventProcessor(db *dbManager, publish *natsManager, events <-chan *Event) {
	for ev := range events {
		var err error
		switch ev.EventID {
		case UserCreatedID:
			err = db.insertUser(ev)
		case PasswordChangedID:
			err = db.updateUserPassword(ev)
		case EmailChangedID:
			err = db.updateUserEmail(ev)
		case EnabledID:
			err = db.enableUser(ev)
		case DisabledID:
			err = db.disableUser(ev)
		default:
			err = fmt.Errorf("unkonwn event while applying: %v", ev.EventID)
		}

		if err != nil {
			log.Println("failed to apply event to database: ", err)
			if publishErr := publish.eventFailed(ev.CorrelationID, []byte(err.Error())); publishErr != nil {
				log.Printf("ERROR: Failed to publish event processing failure: %v (original error: %v)\n", publishErr, err)
			}
		}
		if publishErr := publish.eventSuccess(ev.CorrelationID, []byte(ev.AggregateID)); publishErr != nil {
			log.Printf("ERROR: failed to publish event success message: %v", err)
		}
	}
}

type dbManager struct {
	db *pgxpool.Pool
}

func (m *dbManager) insertUser(ev *Event) error {
	payload := ev.Data.(*UserCreated)
	return m.db.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `
			INSERT INTO users 
				(id, email, password, enabled, last_event_time, last_correlation_id) 
			VALUES ($1, $2, $3, $4, $5, $6)`,
			ev.AggregateID, payload.Email, payload.Password, payload.IsEnabled, ev.CreatedAt, ev.CorrelationID)
		return err
	})
}

func (m *dbManager) updateUserPassword(ev *Event) error {
	payload := ev.Data.(*UserPasswordChanged)
	return m.db.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `UPDATE users SET password=$1 WHERE id=$2`,
			payload.NewPassword, ev.AggregateID)
		return err
	})
}

func (m *dbManager) updateUserEmail(ev *Event) error {
	payload := ev.Data.(*UserEmailChanged)
	return m.db.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `UPDATE users SET email=$1 WHERE id=$2`,
			payload.NewEmail, ev.AggregateID)
		return err
	})
}

func (m *dbManager) enableUser(ev *Event) error {
	return m.db.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `UPDATE users SET enabled=$1 WHERE id=$2`,
			true, ev.AggregateID)
		return err
	})
}

func (m *dbManager) disableUser(ev *Event) error {
	return m.db.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `UPDATE users SET enabled=$1 WHERE id=$2`,
			false, ev.AggregateID)
		return err
	})
}

type natsManager struct {
	conn *nats.Conn
}

func (n *natsManager) eventSuccess(correlationID string, payload []byte) error {
	return n.conn.Publish(fmt.Sprintf("event.%v.success", correlationID), payload)
}

func (n *natsManager) eventFailed(correlationID string, payload []byte) error {
	return n.conn.Publish(fmt.Sprintf("event.%v.error", correlationID), payload)
}
