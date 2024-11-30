package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/lib/pq"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
)

const SQL_DIALECT = "postgres"
const DEFAULT_POSTGRES_PORT int = 5432

// var databases map[string]*sql.DB

var ErrDatabaseAlreadyInitialized = fmt.Errorf("Database connection is already initialized")

// ServiceInitConnectionRequest are the options required to initialize a connection for a service.
// The database name is not required since it's the same as the service name. This is used for
// the generated service code to initialize the database connection for a service.
type ServiceInitConnectionRequest struct {
	Host     string
	Port     int
	User     string
	Password string
	SSLMode  string
}

type Options struct {
	Database string
	Host     string
	Port     int
	User     string
	Password string
	SSLMode  string
}

// QueryableConnection groups together a sql.Connection, sq.DB (with pool handling) and a Transaction
type QueryableConnection interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

var _connectionProviders = map[string]*ConnectionProvider{}
var _connectionProvidersLock = &sync.RWMutex{}

// Connection (DB Connection) is a struct that will hold database connection details
type ConnectionProvider struct {
	Dialect string
	DbName  string
	DB      *sql.DB
	Tx      *sql.Tx
	NumTxs  int
	// UseTransaction bool    // Do we need this extra bool?
	InitOptions Options // Options used to initialize the connection
}

// NewConnectionProvider initializes a totally new database connection provider. It does not check if the connection is already initialized.
// This should ideally not be used directly, but through NewOrExistingConnectionProvider (which reuses existing providers).
func NewConnectionProvider(ctx context.Context, opt Options) (ConnectionProvider, error) {
	prov := ConnectionProvider{
		Dialect:     SQL_DIALECT,
		DbName:      opt.Database,
		InitOptions: opt,
	}

	// Ensure that we are able to connect to the database
	sqlDB, err := prov.GetConnection(ctx)
	if err != nil {
		return ConnectionProvider{}, errutil.Wrap(err, "Could not succesfully test the connection to the database")
	}
	sqlDB.Close()

	return prov, nil
}

func (c *ConnectionProvider) GetConnection(ctx context.Context) (*sql.DB, error) {
	if c.DB != nil {
		if c.DB.Ping() == nil {
			return c.DB, nil
		}
	}

	sqlDB, err := NewSqlConnection(ctx, c.InitOptions)
	if err != nil {
		return nil, err
	}
	c.DB = sqlDB

	return sqlDB, nil
}

// GetSQLConnection returns a direct sql.DB or sql.Tx object that can be used to run queries
func (c ConnectionProvider) GetQueryableConnection(ctx context.Context) (QueryableConnection, error) {
	if c.IsInTransaction(ctx) {
		return c.Tx, nil
	}

	sqlDB, err := c.GetConnection(ctx)
	if err != nil {
		return nil, errutil.Wrap(err, "Could not get a connection to the database")
	}

	return sqlDB, nil
}

// NewOrExistingConnectionProvider initializes a new database connection, and also saves it in the global map of connections.
func NewOrExistingConnectionProvider(ctx context.Context, o Options) (*ConnectionProvider, error) {

	_connectionProvidersLock.Lock()
	defer _connectionProvidersLock.Unlock()

	key, err := getConnectionString(ctx, o)
	if err != nil {
		return nil, errutil.Wrap(err, "geting connection string")
	}

	// Check if the connection is already present
	if connProv, exists := _connectionProviders[key]; exists {
		log.Warn(ctx, "Database connection already initialized", "connStr", key)
		return connProv, nil
	}

	// Otherwise initialize a new connection (and save it)
	prov, err := NewConnectionProvider(ctx, o)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to initialize database")
	}

	_connectionProviders[key] = &prov

	return &prov, nil
}

func GetExistingConnectionProviderByDatabase(ctx context.Context, dbname string) (*ConnectionProvider, error) {
	_connectionProvidersLock.RLock()
	defer _connectionProvidersLock.RUnlock()

	for _, prov := range _connectionProviders {
		if prov.DbName == dbname {
			return prov, nil
		}
	}

	return nil, fmt.Errorf("Database connection for db [%s] not found", dbname)
}

// func GetConnection(ctx context.Context, o Options) (Connection, error) {

// 	// Otherwise initialize a new connection (and save it)
// 	sqlDB, err := GetSQLDB(ctx, o)
// 	if err != nil {
// 		return Connection{}, fmt.Errorf("failed to initialize database: %w", err)
// 	}

// 	return Connection{
// 		DbName:      o.Database,
// 		DB:          sqlDB,
// 		Dialect:     SQL_DIALECT,
// 		InitOptions: o,
// 	}, nil
// }

// func NewConnection(dbname string) (Connection, error) {
// 	db, exists := databases[dbname]
// 	if !exists {
// 		return Connection{}, fmt.Errorf("database connection for db [%s] not found: %w", dbname, errutil.ErrNotFound)
// 	}
// 	conn := Connection{
// 		DbName:  dbname,
// 		DB:      db,
// 		Dialect: SQL_DIALECT,
// 	}
// 	return conn, nil
// }

func NewSqlConnection(ctx context.Context, o Options) (*sql.DB, error) {
	// Create connection string
	connStr, err := getConnectionString(ctx, o)
	if err != nil {
		return nil, err
	}

	log.Debug(ctx, "Initializing SQL connection", "connectionString", connStr)

	// Open the connection
	sqlDB, err := sql.Open(SQL_DIALECT, connStr)
	if err != nil {
		return nil, errutil.Wrap(err, "Could not open the database connection")
	}

	// Test by pinging the database
	err = sqlDB.Ping()
	if err != nil {
		return nil, errutil.Wrap(err, "Could not ping the database using the new connection")
	}

	return sqlDB, nil
}

func getConnectionString(ctx context.Context, o Options) (string, error) {
	if o.Database == "" {
		log.Debug(ctx, "GetConnectionString: Database name not provided. Defaulting to 'postgres'")
		o.Database = "postgres"
	}
	if o.Host == "" {
		return "", fmt.Errorf("GetConnectionString: Host is not provider or is invalid")
	}
	if o.Port < 1 {
		return "", fmt.Errorf("GetConnectionString: Port is not provider or is invalid")
	}
	if o.SSLMode == "" {
		log.Debug(ctx, "GetConnectionString: SSLMode not provided. Defaulting to 'disable'")
		o.SSLMode = "disable"
	}

	str := fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s timezone=%s",
		o.Host, o.Port, o.Database, o.User, o.SSLMode, "UTC")
	if o.Password != "" {
		str += fmt.Sprintf(" password=%s", o.Password)
	}
	return str, nil
}

func (c *ConnectionProvider) Close(ctx context.Context) error {
	// if connection has transactions, rollthem back.
	if c.IsInTransaction(ctx) {
		if err := c.Rollback(ctx); err != nil {
			return err
		}
	}

	// Close the connection (if any)
	if c.DB != nil {
		c.DB.Close()
		c.DB = nil
	}

	return nil
}

// MustClose is usefull for defer function calls where we can't easily handle err returns
func (c *ConnectionProvider) MustClose(ctx context.Context) {
	if err := c.Close(ctx); err != nil {
		panics.IfError(err, "Error closing DB.Connection")
	}

	return
}

func (c *ConnectionProvider) IsInTransaction(ctx context.Context) bool {
	panics.If(c.Tx != nil && c.NumTxs < 1, "Connection.Tx is not nil but number of elements in Connection.TxIDs is zero")
	panics.If(c.Tx == nil && c.NumTxs > 1, "Connection.Tx is nil but number of elements in Connection.TxIDs not zero")
	return c.Tx != nil && c.NumTxs >= 1
}

func (c *ConnectionProvider) Begin(ctx context.Context) error {

	c.NumTxs++

	// If  there is already a transaction, don't do anything
	if c.IsInTransaction(ctx) {
		log.Warn(ctx, "Attempting to begin a nested SQL transaction")
		return nil
	}

	conn, err := c.GetConnection(ctx)
	if err != nil {
		return errutil.Wrap(err, "getting a SQL connection")
	}

	// Otherwise start a transaction
	log.Debug(ctx, "Begin transaction", "number", c.NumTxs)
	txn, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	c.Tx = txn
	// c.UseTransaction = true
	return nil
}

func (c *ConnectionProvider) MustCommit(ctx context.Context) {
	if err := c.Commit(ctx); err != nil {
		panics.IfError(err, "Error committing DB.Transaction")
	}
	return
}

func (c *ConnectionProvider) Commit(ctx context.Context) error {
	if !c.IsInTransaction(ctx) {
		return fmt.Errorf("Attempted to commit a non-transaction")
	}

	c.NumTxs--

	// If we have another deeper transaction left, don't do anything. Only commit on final commit.
	if c.NumTxs > 0 {
		log.Warn(ctx, "Committing a nested SQL transaction. This is a dummy commit, the actual commit will happen on the final commit", "number", c.NumTxs+1)
		return nil
	}
	log.Debug(ctx, "Commiting transaction", "number", c.NumTxs+1)
	err := c.Tx.Commit()
	if err != nil {
		return err
	}
	c.Tx = nil
	// c.UseTransaction = false
	return nil
}

func (c *ConnectionProvider) MustRollback(ctx context.Context) {
	if err := c.Rollback(ctx); err != nil {
		panics.IfError(err, "Error rolling-back DB.Transaction")
	}
	return
}
func (c *ConnectionProvider) Rollback(ctx context.Context) error {
	if !c.IsInTransaction(ctx) {
		return fmt.Errorf("Attempted to rollback a non-transaction")
	}

	// We cannot rollback nested transactions, so can lets rollback everything
	if c.NumTxs > 1 {
		log.Warn(ctx, "Attempting to rollback a nested SQL transaction")
	}
	log.Info(ctx, "Rollback transaction")
	c.NumTxs = 0

	err := c.Tx.Rollback()
	if err != nil {
		return err
	}
	c.Tx = nil
	// c.UseTransaction = false
	return nil
}

// ExecuteQuery executes an insert or update query, returning the number of rows affected.
func (c ConnectionProvider) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (int64, error) {

	log.Debug(ctx, "Executing SQL query", "query", query, "args", args)

	// Execute the Query
	conn, err := c.GetQueryableConnection(ctx)
	if err != nil {
		return -1, errutil.Wrap(err, "failed to get a connection to the database")
	}

	result, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return -1, errutil.Wrap(err, "failed to execute query")
	}

	// Validate the result
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, fmt.Errorf("cannot fetch number of rows affected for executed query: %w", err)
	}
	if rowsAffected < 1 {
		return -1, fmt.Errorf("executed query returned %d rows affected", rowsAffected)
	}

	log.Debug(ctx, "Query successfully executed", "rowsAffected", rowsAffected)

	return rowsAffected, nil
}

// QueryRows executes a query that returns rows e.g. Select Query
func (c ConnectionProvider) QueryRows(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {

	log.Debug(ctx, "SQL Query for fetch", "query", query, "args", args)

	// Get the connection that we'll use to execute the query. This `conn` can be a direct DB connection or a transaction
	conn, err := c.GetQueryableConnection(ctx)
	if err != nil {
		return nil, errutil.Wrap(err, "failed to get a connection to the database")
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	log.Debug(ctx, "Query ran successfully.")

	return rows, nil
}

func (c ConnectionProvider) QueryRow(ctx context.Context, v interface{}, query string, args ...interface{}) error {

	log.Debug(ctx, "SQL Query for fetch", "query", query, "args", args)

	conn, err := c.GetQueryableConnection(ctx)
	if err != nil {
		return errutil.Wrap(err, "failed to get a connection to the database")
	}

	row := conn.QueryRowContext(ctx, query, args...)
	err = row.Err()
	if err != nil {
		log.Debug(ctx, "Error quering row", "query", query, "args", args, "error", err)
		return row.Err()

	}

	err = row.Scan(v)
	if err != nil {
		log.Debug(ctx, "Error scanning row", "query", query, "args", args, "error", err)
		return err
	}

	return nil
}
