package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/scalars"
)

type InsertBuilderRequest struct {
	TableName   string
	ColumnNames []string
	Values      [][]interface{}
}

func ConstructInsertQuery(ctx context.Context, dialectStr string, req InsertBuilderRequest) (string, []interface{}, error) {
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
	dialect := goqu.Dialect(dialectStr)

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

type ConstructUpsertQueryRequest struct {
	Dialect      string
	TableName    string
	UpsertColumn string // The column that is the primary key, used to check if the row exists
	ColumnNames  []string
	Values       []interface{}
}

func ConstructUpsertQuery(ctx context.Context, req ConstructUpsertQueryRequest) (string, []interface{}, error) {
	log.Debug(ctx, "Constructing upsert query", "request", PrettyPrint(req))

	// Validate
	if len(req.ColumnNames) < 1 {
		return "", nil, fmt.Errorf("no column names provided")
	}
	if len(req.Values) < 1 {
		return "", nil, fmt.Errorf("no values provided")
	}
	if len(req.ColumnNames) != len(req.Values) {
		return "", nil, fmt.Errorf("number of values [%d] does not match the number of columns [%d]", len(req.Values), len(req.ColumnNames))
	}

	// Get the value of the upsert column
	upsertColumnIndex := -1
	for i, col := range req.ColumnNames {
		if col == req.UpsertColumn {
			upsertColumnIndex = i
			break
		}
	}
	if upsertColumnIndex == -1 {
		return "", nil, fmt.Errorf("upsert column [%s] not found in the column names [%v]", req.UpsertColumn, req.ColumnNames)
	}
	upsertValue := req.Values[upsertColumnIndex]

	// Construct the query
	dialect := goqu.Dialect(req.Dialect)

	// Goqu does not have a built-in upsert function, so we will have to check if the row exists and then insert or update
	// Check if the row exists
	cnt, err := dialect.From(req.TableName).Where(goqu.C(req.UpsertColumn).Eq(upsertValue)).Count()
	if err != nil {
		return "", nil, errutil.Wrap(err, "Creating 'count' query to check if the row exists")
	}

	// If the row exists, update it
	if cnt > 0 {
		// Construct the update query
		query, args, err := ConstructUpdateQuery(ctx, req.Dialect, UpdateBuilderRequest{
			TableName:        req.TableName,
			Columns:          req.ColumnNames,
			Values:           req.Values,
			IdentifierColumn: req.UpsertColumn,
			IdentifierValue:  upsertValue,
		})
		if err != nil {
			return "", nil, errutil.Wrap(err, "Constructing update query")
		}
		return query, args, nil
	}

	// If the row does not exist, insert it
	query, args, err := ConstructInsertQuery(ctx, req.Dialect, InsertBuilderRequest{
		TableName:   req.TableName,
		ColumnNames: req.ColumnNames,
		Values:      [][]interface{}{req.Values},
	})

	if err != nil {
		return "", nil, errutil.Wrap(err, "Constructing insert query")
	}
	return query, args, nil
}

type UpdateBuilderRequest struct {
	TableName string
	Columns   []string
	Values    []interface{}
	// For where condition, so we only update the required row
	IdentifierColumn string
	IdentifierValue  interface{}
}

// ConstructSelectQuery creates a string SQL query with args
func ConstructUpdateQuery(ctx context.Context, dialectStr string, req UpdateBuilderRequest) (string, []interface{}, error) {

	log.Debug(ctx, "Constructing query for update", "request", PrettyPrint(req))

	// Validate
	if len(req.Columns) < 1 {
		return "", nil, fmt.Errorf("needs at least one Column, got none")
	}
	if len(req.Columns) != len(req.Values) {
		return "", nil, fmt.Errorf("expects the number of values (%d) to match the number of columns (%d)", len(req.Values), len(req.Columns))
	}

	// DRY this query generation part to db package
	dialect := goqu.Dialect(dialectStr)

	// Create map to represent the update set
	var mp = make(map[string]interface{})
	for i := 0; i < len(req.Columns); i++ {
		mp[req.Columns[i]] = req.Values[i]
	}

	// Query: Select Part
	ds := dialect.Update(req.TableName).
		Set(mp).
		Where(goqu.C(req.IdentifierColumn).Eq(req.IdentifierValue))

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
func ConstructSelectByIDQuery(ctx context.Context, dialectStr string, req SelectByIDBuilderRequest) (string, []interface{}, error) {

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
	ds := goqu.Dialect(dialectStr).
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
	panics.IfError(err, "Cannot json.MarshalIndent: %s\ndata: %+v", err, i)
	return string(s)
}
