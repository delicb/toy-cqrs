package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
)

// DBManager describes database operations needed by this service.
type DBManager interface {
	// GetUser returns instance of a user model from the database.
	GetUser(id string) (*UserModel, error)
}

type dbManager struct {
	db *pgx.Conn
}

// NewDBManager returns instance of a DB manager.
func NewDBManager(ctx context.Context, dsn string) *dbManager {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		panic(err)
	}
	return &dbManager{conn}
}

func (d *dbManager) GetUser(id string) (*UserModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	row := d.db.QueryRow(ctx,
		`SELECT id, email, enabled FROM users WHERE id = $1`,
		id,
	)
	u := &UserModel{}
	return u, row.Scan(&u.ID, &u.Email, &u.Enabled)
}
