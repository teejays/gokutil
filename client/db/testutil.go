package db

import (
	"context"

	_ "github.com/lib/pq"
	"github.com/teejays/gokutil/errutil"
)

func GetTestDatabaseNameForService(serviceName string) string {
	return serviceName + "_test"
}

func NewMockConnection(ctx context.Context, serviceName string) (*Connection, error) {
	dbName := GetTestDatabaseNameForService(serviceName)

	connProv, err := NewConnectionProvider(ctx, Options{
		Database: dbName,
		ServerOptions: ServerOptions{
			Host:    "localhost",
			Port:    DEFAULT_POSTGRES_PORT,
			User:    "tansari",
			SSLMode: "disable",
		},
	})
	if err != nil {
		return nil, errutil.Wrap(err, "Creating new connection provider")
	}

	conn, err := connProv.GetConnection(ctx)
	if err != nil {
		return nil, errutil.Wrap(err, "Getting connection")
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
