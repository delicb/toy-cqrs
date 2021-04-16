package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/delicb/toy-cqrs/types"
)

type EventPersistenceStore interface {
	// GetEvents returns all events for provided aggregate ID sorted by creation time.
	GetEvents(id string) ([]types.Event, error)

	// SaveEvents persists all provided events.
	SaveEvents([]types.Event) error
}

type dbManager struct {
	conn *pgx.Conn
}

func newDBManager(dsn string) (*dbManager, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &dbManager{conn}, nil
}

func (db *dbManager) GetEvents(id string) ([]types.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rows, err := db.conn.Query(ctx,
		`SELECT aggregate_id, created_at, correlation_id, event_id, payload 
			FROM events 
			WHERE aggregate_id = $1 
			ORDER BY created_at ASC`, id)
	if err != nil {
		return nil, err
	}
	events := make([]types.Event, 0)
	for rows.Next() {
		var aggregateID string
		var createdAt time.Time
		var correlationID string
		var eventID string
		var payload []byte
		if err := rows.Scan(&aggregateID, &createdAt, &correlationID, &eventID, &payload); err != nil {
			return nil, err
		}
		ev, err := types.NewEvent(types.EventID(eventID), aggregateID, correlationID, nil)
		if err != nil {
			return nil, err
		}
		ev.SetRawPayload(payload)
		events = append(events, ev)
	}
	for _, ev := range events {
		log.Printf("got event from DB: %v for %v", ev.ID(), ev.AggregateID())
	}
	return events, nil
}

func (db *dbManager) SaveEvents(events []types.Event) (err error) {
	log.Println("saving events to database")

	return db.conn.BeginFunc(context.Background(), func(tx pgx.Tx) error {
		for _, ev := range events {
			log.Printf("saving %d events to the database\n", len(events))
			_, err = tx.Exec(context.Background(), `
			INSERT INTO events
				(aggregate_id, created_at, correlation_id, event_id, payload)
			VALUES
				($1, $2, $3, $4, $5)`,
				ev.AggregateID(), ev.CreatedAt(), ev.CorrelationID(), ev.ID(), ev.RawPayload())

			if err != nil {
				return err
			}
		}
		return nil
	})
}

var _ EventPersistenceStore = &dbManager{}
