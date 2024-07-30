package db

import (
	"context"
	"errors"

	_ "github.com/lib/pq"
	"github.com/teejays/clog"
)

func GetTestDatabaseNameForService(serviceName string) string {
	return serviceName + "_test"
}

func NewMockConnection(ctx context.Context, serviceName string) (Connection, error) {
	var conn Connection
	dbName := GetTestDatabaseNameForService(serviceName)
	err := InitDatabase(ctx, Options{
		Host:     "localhost",
		Port:     DEFAULT_POSTGRES_PORT,
		Database: dbName,
		User:     "tansari",
		SSLMode:  "disable",
	})
	if err != nil && !errors.Is(err, ErrDatabaseAlreadyInitialized) {
		clog.Warnf("Database %s already initialized", dbName)
		return conn, err
	}

	conn, err = NewConnection(dbName)
	if err != nil {
		return conn, err
	}

	err = conn.Begin(ctx)
	if err != nil {
		return conn, err
	}
	// Question: What happens if there is a conectionName conflict? aka. two or more
	// independent tests use the same connectionName and run in parallel

	// TODO: we need to defer Close() the connection
	return conn, nil
}
