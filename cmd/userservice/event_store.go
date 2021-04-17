package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v4"
)

// EventStore is description of persistence for events.
type EventStore interface {
	// Load returns all events for provided aggregate root id.
	Load(aggregateID string) ([]*Event, error)

	// Save persist all provided events.
	Save([]*Event) error
}

// inMemoryStore is simple implementation of storage that does not persist
// events, but keeps them in memory instead.
type inMemoryStore struct {
	state map[string][]*Event
}

func (s *inMemoryStore) Load(aggregateID string) ([]*Event, error) {
	return s.state[aggregateID], nil
}

func (s *inMemoryStore) Save(events []*Event) error {
	log.Printf("saving events: %+v", events)
	for _, ev := range events {
		s.state[ev.AggregateID] = append(s.state[ev.AggregateID], ev)
	}
	return nil
}

// NewInMemoryEventStore returns EventStore implementation that stores events only in memory.
func NewInMemoryEventStore() *inMemoryStore {
	return &inMemoryStore{
		state: make(map[string][]*Event),
	}
}

type psqlEventStorage struct {
	conn *pgx.Conn
}

func NewPsqlEventStorage(ctx context.Context, dsn string) (*psqlEventStorage, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &psqlEventStorage{conn}, nil
}

func (p *psqlEventStorage) Load(aggregateID string) ([]*Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rows, err := p.conn.Query(ctx,
		`SELECT aggregate_id, created_at, correlation_id, event_id, payload
			FROM events
			WHERE aggregate_id = $1
			ORDER BY created_at ASC`, aggregateID)
	if err != nil {
		return nil, err
	}
	events := make([]*Event, 0)
	for rows.Next() {
		var aggregateID string
		var createdAt time.Time
		var correlationID string
		var eventID EventID
		var payload []byte
		if err := rows.Scan(&aggregateID, &createdAt, &correlationID, &eventID, &payload); err != nil {
			return nil, err
		}
		data, err := UnmarshalEventData(eventID, payload)
		if err != nil {
			return nil, err
		}
		ev := &Event{
			EventID:       eventID,
			AggregateID:   aggregateID,
			CreatedAt:     createdAt,
			CorrelationID: correlationID,
			Data:          data,
		}
		events = append(events, ev)
	}
	return events, nil
}

func (p *psqlEventStorage) Save(events []*Event) error {
	log.Println("saving events to the database")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return p.conn.BeginFunc(ctx, func(tx pgx.Tx) error {
		for _, ev := range events {
			payload, err := json.Marshal(ev.Data)
			if err != nil {
				return err
			}
			_, err = tx.Exec(context.Background(), `
				INSERT INTO events
					(aggregate_id, created_at, correlation_id, event_id, payload)
				VALUES
					($1, $2, $3, $4, $5)`,
				ev.AggregateID, ev.CreatedAt, ev.CorrelationID, ev.EventID, payload,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
