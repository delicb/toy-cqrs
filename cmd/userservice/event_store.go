package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/delicb/toy-cqrs/cqrs"
	"github.com/delicb/toy-cqrs/users"
)

type psqlEventStorage struct {
	conn           *pgx.Conn
	afterSaveHooks []cqrs.EventHook
}

// NewPsqlEventStore implements cqrs.EventStore interface on top of Postgres database.
func NewPsqlEventStore(ctx context.Context, dsn string) (*psqlEventStorage, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &psqlEventStorage{
		conn:           conn,
		afterSaveHooks: make([]cqrs.EventHook, 0),
	}, nil
}

func (p *psqlEventStorage) Load(aggregateID string) ([]*cqrs.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	rows, err := p.conn.Query(ctx,
		`SELECT aggregate_id, aggregate_type, created_at, correlation_id, event_id, data
			FROM events
			WHERE aggregate_id = $1
			ORDER BY created_at ASC`, aggregateID)
	if err != nil {
		return nil, err
	}
	return rowsToEvents(rows)
}

func (p *psqlEventStorage) Save(events []*cqrs.Event) error {
	log.Println("saving events to the database")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	txErr := p.conn.BeginFunc(ctx, func(tx pgx.Tx) error {
		for _, ev := range events {
			data, err := users.EventSerializer.MarshalData(ev)
			if err != nil {
				return err
			}
			_, err = tx.Exec(context.Background(), `
				INSERT INTO events
					(aggregate_id, aggregate_type, created_at, correlation_id, event_id, data)
				VALUES
					($1, $2, $3, $4, $5, $6)`,
				ev.AggregateID, ev.AggregateType, ev.CreatedAt, ev.CorrelationID, ev.EventID, data,
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}

	// call save hooks
	for _, ev := range events {
		for _, h := range p.afterSaveHooks {
			h(ev)
		}
	}

	return nil
}

func (p *psqlEventStorage) LoadEmailEvents() ([]*cqrs.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	rows, err := p.conn.Query(ctx, `
		SELECT aggregate_id, created_at, correlation_id, event_id, data 
		FROM events 
		WHERE 
			event_id='user.created' OR 
			event_id='user.email.changed' 
		ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}

	return rowsToEvents(rows)
}

// AddAfterSaveHook add a function to be called when event is saved.
func (p *psqlEventStorage) AddAfterSaveHook(h cqrs.EventHook) {
	p.afterSaveHooks = append(p.afterSaveHooks, h)
}

func rowsToEvents(rows pgx.Rows) ([]*cqrs.Event, error) {
	events := make([]*cqrs.Event, 0)
	for rows.Next() {
		var aggregateID string
		var aggregateType string
		var createdAt time.Time
		var correlationID string
		var eventID cqrs.EventID
		var data []byte
		if err := rows.Scan(&aggregateID, &aggregateType, &createdAt, &correlationID, &eventID, &data); err != nil {
			return nil, err
		}
		eventData, err := users.EventSerializer.UnmarshalData(eventID, data)
		if err != nil {
			return nil, err
		}
		ev := &cqrs.Event{
			EventID:       eventID,
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			CreatedAt:     createdAt,
			CorrelationID: correlationID,
			Data:          eventData,
		}
		events = append(events, ev)
	}
	return events, nil
}
