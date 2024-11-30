package db

import (
	"context"

	_ "github.com/lib/pq"
	"github.com/teejays/gokutil/errutil"
)

func GetTestDatabaseNameForService(serviceName string) string {
	return serviceName + "_test"
}

func NewMockConnection(ctx context.Context, serviceName string) (ConnectionProvider, error) {
	dbName := GetTestDatabaseNameForService(serviceName)

	connProv, err := NewConnectionProvider(ctx, Options{
		Host:     "localhost",
		Port:     DEFAULT_POSTGRES_PORT,
		Database: dbName,
		User:     "tansari",
		SSLMode:  "disable",
	})
	if err != nil {
		return connProv, errutil.Wrap(err, "Creating new connection provider")
	}

	err = connProv.Begin(ctx)
	if err != nil {
		return connProv, err
	}
	// Question: What happens if there is a conectionName conflict? aka. two or more
	// independent tests use the same connectionName and run in parallel

	// TODO: we need to defer Close() the connection
	return connProv, nil
}
