package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/lib/pq"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/scalars"
)

const SQL_DIALECT = "postgres"
const DEFAULT_POSTGRES_PORT int = 5432

var databases map[string]*sql.DB

var ErrDatabaseAlreadyInitialized = fmt.Errorf("Database connection is already initialized")

// ServiceInitConnectionRequest are the options required to initialize a connection for a service.
// The database name is not required since it's the same as the service name. This is used for
// the generated service code to initialize the database connection for a service.
type ServiceInitConnectionRequest struct {
	Host     string
	Port     int
	User     string
	Password string
}

type Options struct {
	Database string
	Host     string
	Port     int
	User     string
	Password string
	SSLMode  string
}

func InitDatabase(ctx context.Context, o Options) error {
	db, err := GetDB(ctx, o)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	databases[o.Database] = db
	return nil
}

func GetConnection(ctx context.Context, o Options) (Connection, error) {
	// If an existing connection is already present, return it
	conn, ok := databases[o.Database]
	if ok {
		return Connection{
			DbName: o.Database,
			DB:     conn,
		}, nil
	}

	// Otherwise initialize a new connection (and save it)
	db, err := GetDB(ctx, o)
	if err != nil {
		return Connection{}, fmt.Errorf("failed to initialize database: %w", err)
	}

	databases[o.Database] = db
	return Connection{
		DbName: o.Database,
		DB:     db,
	}, nil
}

func GetDB(ctx context.Context, o Options) (*sql.DB, error) {
	connStr, err := getConnectionString(o)
	if err != nil {
		return nil, err
	}

	log.Debug(ctx, "Initializing SQL connection", "connectionString", connStr)

	if databases == nil {
		databases = make(map[string]*sql.DB)
	}

	if databases[o.Database] != nil {
		return nil, fmt.Errorf("database '%s': %w", o.Database, ErrDatabaseAlreadyInitialized)
	}

	db, err := sql.Open(SQL_DIALECT, connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("DB connection ping failed: %w", err)
	}

	return db, nil
}

func getConnectionString(o Options) (string, error) {
	if o.Database == "" {
		log.DebugWithoutCtx("GetConnectionString: Database name not provided. Defaulting to 'postgres'")
		o.Database = "postgres"
	}
	if o.Host == "" {
		return "", fmt.Errorf("GetConnectionString: Host is not provider or is invalid")
	}
	if o.Port < 1 {
		return "", fmt.Errorf("GetConnectionString: Port is not provider or is invalid")
	}
	if o.SSLMode == "" {
		log.DebugWithoutCtx("GetConnectionString: SSLMode not provided. Defaulting to 'disable'")
		o.SSLMode = "disable"
	}

	str := fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s timezone=%s",
		o.Host, o.Port, o.Database, o.User, o.SSLMode, "UTC")
	if o.Password != "" {
		str += fmt.Sprintf(" password=%s", o.Password)
	}
	return str, nil
}

// CheckConnection ensures that the DB connection is established and working
func CheckConnection(dbname string) error {
	conn, err := NewConnection(dbname)
	if err != nil {
		return err
	}
	err = conn.DB.Ping()
	if err != nil {
		return fmt.Errorf("DB connection ping failed: %w", err)
	}
	return nil
}

// Connection (DB Connection) is a struct that will hold database connection details
type Connection struct {
	Dialect        string
	DbName         string
	DB             *sql.DB
	Tx             *sql.Tx
	NumTxs         int
	UseTransaction bool // Do we need this extra bool?
}

func NewConnection(dbname string) (Connection, error) {
	db, exists := databases[dbname]
	if !exists {
		return Connection{}, fmt.Errorf("database connection for db [%s] not found: %w", dbname, errutil.ErrNotFound)
	}
	conn := Connection{
		DbName:  dbname,
		DB:      db,
		Dialect: SQL_DIALECT,
	}
	return conn, nil
}

func (c *Connection) Close(ctx context.Context) error {
	// if connection has transactions, rollthem back.
	if c.IsInTransaction(ctx) {
		if err := c.Rollback(ctx); err != nil {
			return err
		}
	}

	return c.DB.Close()
}

// MustClose is usefull for defer function calls where we can't easily handle err returns
func (c *Connection) MustClose(ctx context.Context) {
	if err := c.Close(ctx); err != nil {
		panics.IfError(err, "Error closing DB.Connection")
	}

	return
}

func (c *Connection) IsInTransaction(ctx context.Context) bool {
	panics.If(c.Tx != nil && c.NumTxs < 1, "Connection.Tx is not nil but number of elements in Connection.TxIDs is zero")
	panics.If(c.Tx == nil && c.NumTxs > 1, "Connection.Tx is nil but number of elements in Connection.TxIDs not zero")
	return c.Tx != nil && c.NumTxs >= 1
}

func (c *Connection) Begin(ctx context.Context) error {

	c.NumTxs++

	// If  there is already a transaction, don't do anything
	if c.IsInTransaction(ctx) {
		log.Warn(ctx, "Attempting to begin a nested SQL transaction")
		return nil
	}

	// Otherwise start a transaction
	log.Info(ctx, "Begin transaction")
	txn, err := c.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	c.Tx = txn
	c.UseTransaction = true
	return nil
}

func (c *Connection) MustCommit(ctx context.Context) {
	if err := c.Commit(ctx); err != nil {
		panics.IfError(err, "Error committing DB.Transaction")
	}
	return
}

func (c *Connection) Commit(ctx context.Context) error {
	if !c.IsInTransaction(ctx) {
		return fmt.Errorf("Attempted to commit a non-transaction")
	}

	c.NumTxs--

	// If we have another deeper transaction left, don't do anything. Only commit on final commit.
	if c.NumTxs > 0 {
		log.Warn(ctx, "Attempting to commit a nested SQL transaction")
		return nil
	}
	log.Info(ctx, "Commiting transaction")
	err := c.Tx.Commit()
	if err != nil {
		return err
	}
	c.Tx = nil
	c.UseTransaction = false
	return nil
}

func (c *Connection) MustRollback(ctx context.Context) {
	if err := c.Rollback(ctx); err != nil {
		panics.IfError(err, "Error rolling-back DB.Transaction")
	}
	return
}
func (c *Connection) Rollback(ctx context.Context) error {
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
	c.UseTransaction = false
	return nil
}

// SQLConnection groups together a sql.Connection, sq.DB (with pool handling) and a Transaction
type SQLConnection interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// GetSQLConnection returns a direct sql.DB or sql.Tx object that can be used to run queries
func (c Connection) GetSQLConnection() SQLConnection {
	if c.UseTransaction {
		panics.IfNil(c.Tx, "Connection's transaction is nil but UseTransaction set to true")
		return c.Tx
	}
	return c.DB
}

// ExecuteQuery executes an insert or update query, returning the number of rows affected.
func (c Connection) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (int64, error) {

	log.Debug(ctx, "Executing SQL query", "query", query, "args", args)

	// Execute the Query
	conn := c.GetSQLConnection()

	result, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return -1, fmt.Errorf("failed to execute query: %w", err)
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
func (c Connection) QueryRows(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {

	log.Debug(ctx, "SQL Query for fetch", "query", query, "args", args)

	// Get the connection that we'll use to execute the query. This `conn` can be a direct DB connection or a transaction
	conn := c.GetSQLConnection()

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	log.Debug(ctx, "Query ran successfully.")

	return rows, nil
}

type InsertBuilderRequest struct {
	TableName   string
	ColumnNames []string
	Values      [][]interface{}
}

func (c Connection) ConstructInsertQuery(ctx context.Context, req InsertBuilderRequest) (string, []interface{}, error) {
	log.Debug(ctx, "Constructing query for insert", "request", PrettyPrint(req))

	// Validate
	if len(req.ColumnNames) < 1 {
		return "", nil, fmt.Errorf("no column names provided")
	}
	if len(req.Values) < 1 {
		return "", nil, fmt.Errorf("no values provided")
	}
	for i, vals := range req.Values {
		if len(vals) < 1 {
			return "", nil, fmt.Errorf("value #%d is has no elems i.e. is empty", i+1)
		}
		if len(req.ColumnNames) != len(vals) {
			return "", nil, fmt.Errorf("number of elems for the value #%d is not equal to the number of columns", i+1)
		}
	}

	// Construct the query
	dialect := goqu.Dialect(c.Dialect)

	// Add Table Name
	ds := dialect.Insert(req.TableName)

	// Add Columns
	var colNamesI []interface{}
	for i := range req.ColumnNames {
		colNamesI = append(colNamesI, req.ColumnNames[i])
	}
	ds = ds.Cols(colNamesI...)

	// Add Values
	ds = ds.Vals(req.Values...)

	// Construct Query
	query, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

type UpdateBuilderRequest struct {
	TableName string
	Columns   []string
	Values    []interface{}
	// For where condition, so we only update the required row
	ID scalars.ID
}

// ConstructSelectQuery creates a string SQL query with args
func (c Connection) ConstructUpdateQuery(ctx context.Context, req UpdateBuilderRequest) (string, []interface{}, error) {

	log.Debug(ctx, "Constructing query for update", "request", PrettyPrint(req))

	// Validate
	if len(req.Columns) < 1 {
		return "", nil, fmt.Errorf("needs at least one Column, got none")
	}
	if len(req.Columns) != len(req.Values) {
		return "", nil, fmt.Errorf("expects the number of values (%d) to match the number of columns (%d)", len(req.Values), len(req.Columns))
	}

	// DRY this query generation part to db package
	dialect := goqu.Dialect(c.Dialect)

	// Create map to represent the update set
	var mp = make(map[string]interface{})
	for i := 0; i < len(req.Columns); i++ {
		mp[req.Columns[i]] = req.Values[i]
	}

	// Query: Select Part
	ds := dialect.Update(req.TableName).
		Set(mp).
		Where(goqu.C("id").Eq(req.ID))

	// Fetch the main entities
	query, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}

	return query, args, nil
}

type SelectByIDBuilderRequest struct {
	TableName string
	Columns   []string
	// For where condition, so we only update the required row
	IDColumn string
	IDs      []scalars.ID
}

// ConstructSelectQuery creates a string SQL query with args
func (c Connection) ConstructSelectByIDQuery(ctx context.Context, req SelectByIDBuilderRequest) (string, []interface{}, error) {

	// Validate
	if len(req.Columns) < 1 {
		return "", nil, fmt.Errorf("needs at least one Column, got none")
	}
	if req.IDColumn == "" {
		return "", nil, fmt.Errorf("no ID column name provided")
	}
	if len(req.IDs) < 1 {
		return "", nil, fmt.Errorf("needs at least one ID value, got none")
	}

	// Construct the query
	ds := goqu.Dialect(c.Dialect).
		From(req.TableName)

	ds = ds.Select(StringsToInterfaces(req.Columns)...)

	ds = ds.Where(
		goqu.C(req.IDColumn).In(UUIDsToInterfaces(req.IDs)...),
	)

	query, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}

	return query, args, nil
}

type InsertTypeParams struct {
	TableName string
}

type ListTypeByIDsParams struct {
	TableName string
	IDColumn  string
	IDs       []scalars.ID
}

type UniqueIDsQueryBuilderParams struct {
	TableName     string
	SelectColumns []string
}

// Insert Helpers

func UUIDsToInterfaces(ids []scalars.ID) []interface{} {
	var r []interface{}
	for _, id := range ids {
		r = append(r, id)
	}
	return r
}

func StringsToInterfaces(strs []string) []interface{} {
	var r []interface{}
	for _, s := range strs {
		r = append(r, s)
	}
	return r
}

type SelectQueryBuilder struct {
	Withs  []With
	Select *goqu.SelectDataset
}

type With struct {
	Alias  string
	Select *goqu.SelectDataset
}

func (b SelectQueryBuilder) ToGoquDataset() *goqu.SelectDataset {
	ds := b.Select

	for _, with := range b.Withs {
		ds = ds.With(with.Alias, with.Select)
	}
	return ds
}

func (b SelectQueryBuilder) ToSQL() (string, []interface{}, error) {
	// Contstruct a Goqu.SelectDataset from SelectQueryBuilder
	ds := b.Select
	for _, with := range b.Withs {
		ds = ds.With(with.Alias, with.Select)
	}

	// Convert to query
	query, args, err := ds.ToSQL()
	if err != nil {
		return "", nil, err
	}

	return query, args, nil
}

func SqlRowsToUUIDs(ctx context.Context, rows *sql.Rows) ([]scalars.ID, error) {
	var ids []scalars.ID
	for rows.Next() {
		var id scalars.ID
		err := rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("sql.Row scan error: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func PrettyPrint(i interface{}) string {
	s, err := json.MarshalIndent(i, "", "\t")
	panics.IfError(err, "Cannot json.MarshalIndent: %s", err)
	return string(s)
}

// // InitAndTestConnectionForDb initializes connections for the given database
// func InitAndTestConnectionForDb(ctx context.Context, dbName string) error {
// 	var err error

// 	// Ensure neccessary ENV variables are set
// 	if os.Getenv("DATABASE_HOST") == "" {
// 		return fmt.Errorf("ENV variable DATABASE_HOST not set")
// 	}
// 	if os.Getenv("POSTGRES_USERNAME") == "" {
// 		return fmt.Errorf("ENV variable POSTGRES_USERNAME not set")
// 	}
// 	if os.Getenv("POSTGRES_PASSWORD") == "" {
// 		log.Warn(ctx, "Initializing Database Connection: ENV variable POSTGRES_PASSWORD not set")
// 	}

// 	port := DEFAULT_POSTGRES_PORT
// 	portStr := os.Getenv("DATABASE_PORT")
// 	if portStr != "" {
// 		port, err = strconv.Atoi(portStr)
// 		if err != nil {
// 			return fmt.Errorf("Env variable [DATABASE_PORT] value [%s] is not a number: %w", portStr, err)
// 		}
// 	}
// 	log.Debug(ctx, "Initializing database connection", "database", dbName)
// 	err = InitDatabase(ctx, Options{
// 		Host:     os.Getenv("DATABASE_HOST"),
// 		Port:     port,
// 		Database: dbName,
// 		User:     os.Getenv("POSTGRES_USERNAME"),
// 		Password: os.Getenv("POSTGRES_PASSWORD"),
// 		SSLMode:  "disable",
// 	})
// 	if err != nil {
// 		return fmt.Errorf("Initalizing database %w", err)
// 	}

// 	return nil
// }
