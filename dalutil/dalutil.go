package dalutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/teejays/gokutil/log"

	"github.com/teejays/gokutil/client/db"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/naam"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/scalars"
	"github.com/teejays/gokutil/types"
	"github.com/teejays/gokutil/validate"
)

var llog = log.GetLogger().WithHeading("DALUtil")

type IEntityDALMeta[T types.EntityType, F types.Field] interface {
	types.IEntityMeta[T, F]
	GetDbTableName() naam.Name
	GetTypeDALMeta() ITypeDALMeta[T, F]
}

// EntityDALMeta is holds knowledge for DAL Meta for an entity.
type EntityCommonDALMeta[T types.BasicType, F types.Field] struct {
	DbTableName naam.Name
	TypeDALMeta ITypeDALMeta[T, F]
}

func (m EntityCommonDALMeta[T, F]) GetDbTableName() naam.Name {
	return m.DbTableName
}

func (m EntityCommonDALMeta[T, F]) GetTypeDALMeta() ITypeDALMeta[T, F] {
	return m.TypeDALMeta
}

// ITypeDALMeta is the interface that provides the basic methods that a DAL Meta for a type to implement.
type ITypeDALMeta[T types.BasicType, F types.Field] interface {
	types.ITypeMeta[T, F]
	GetCommonDALMeta() *TypeCommonDALMeta[T, F]

	GetDatabaseColumns() []string
	// Add
	GetDirectDBValues(T) []interface{} // Returns all the values that need to be added to DB
	AddSubTableFieldsToDB(context.Context, db.Connection, db.InsertTypeParams, T) (T, error)
	// ListType
	ScanDBNextRow(*sql.Rows) (T, error)
	FetchSubTableFields(context.Context, db.Connection, db.ListTypeByIDsParams, []T) ([]T, error)
	// Update Type
	GetChangedFieldsAndValues(old, new T, allowedFields []F) ([]F, []interface{})
	UpdateSubTableFields(context.Context, db.Connection, UpdateTypeRequest[T, F], []F, T) (T, error) // TODO

}

// TypeCommonDALMeta
type TypeCommonDALMeta[T types.BasicType, F types.Field] struct {
	DatabaseColumnFields          []F // fields that are direct SQL Columns
	DatabaseSubTableFields        []F
	DatabaseColumnTimestampFields []F // fields that are direct DB columns, not repeated, and timestamps
	SetInternallyByDALFields      []F
	ImmutableFields               []F // Fields that can never be updated one object has been created
	UpdatedAtField                F
}

//	func NewBasicTypeDALMetaBase[T types.BasicType, F types.Field]() ITypeDALMeta[T, F] {
//		return &TypeCommonDALMeta[T, F]{}
//	}
func (m TypeCommonDALMeta[T, F]) GetDatabaseColumns() []string {
	return FieldsToStrings(m.DatabaseColumnFields)
}

type EntityAddRequest[T any] struct {
	Object T `json:"object"`
}

type EntityBatchAddRequest[T any] struct {
	Objects []T `json:"objects"`
}

type TypeBatchAddRequest[T any] struct {
	Objects []T `json:"objects"`
}

// type AddTypeResponse[T types.BasicType] struct {
// 	Object T
// }

type GetEntityRequest[T types.BasicType] struct {
	GetTypeRequest[T]
}

type GetTypeRequest[T types.BasicType] struct {
	ID scalars.ID
}

type ListEntityRequest[T types.FilterType] struct {
	Filter T `json:"filter"`
}

type ListEntityResponse[T types.BasicType] struct {
	Items []T `json:"items"`
	Count int `json:"count"`
}

type ListTypeRequest[T types.FilterType] struct {
	Filter T `json:"filter"`
}

type ListTypeResponse[T types.BasicType] struct {
	Items []T `json:"items"`
	Count int `json:"count"`
	// MAYBE TODO: PageInfo interface{}
}

// Todo: Make QueryByText part of the List methods, by including a Query field in the filters
type QueryByTextEntityRequest[T types.BasicType] struct {
	QueryByTextTypeRequest[T]
}

type QueryByTextTypeRequest[T types.BasicType] struct {
	QueryText string `json:"query_text"`
}

type UpdateEntityRequest[T types.BasicType, F types.Field] struct {
	Object        T   `json:"object"`
	Fields        []F `json:"fields"`
	ExcludeFields []F `json:"excludeFields"`
}

type UpdateEntityResponse[T types.BasicType] struct {
	Object T
}

type UpdateTypeRequest[T types.BasicType, F types.Field] struct {
	TableName     string
	Object        T
	Fields        []F
	ExcludeFields []F
}

type UpdateTypeResponse[T types.BasicType] struct {
	Object T
}

func FieldsToStrings[T types.Field](fields []T) []string {
	var colNames []string
	for _, f := range fields {
		colNames = append(colNames, f.Name().FormatSQL())
	}
	return colNames
}

func BatchAddType[T types.BasicType, F types.Field](ctx context.Context, conn db.Connection, params db.InsertTypeParams, meta ITypeDALMeta[T, F], elems ...T) ([]T, error) {

	now := time.Now()

	// Meta info

	// Set the Meta field values for each elem
	for i := range elems {
		elems[i] = meta.SetMetaFieldValues(ctx, elems[i], now)
	}

	// Convert all timestamps to UTC
	for i := range elems {
		elems[i] = meta.ConvertTimestampColumnsToUTC(elems[i])
	}

	// Add any default values for missing fields
	for i := range elems {
		elems[i] = meta.SetDefaultFieldValues(elems[i])
	}

	// Run any before create hooks
	if fn := meta.GetHookAddBefore(); fn != nil {
		log.Warn(ctx, "Running BeforeCreateHook", "type", meta.GetTypeCommonMeta().Name)
		for i := range elems {
			var err error
			elems[i], err = fn(ctx, elems[i])
			if err != nil {
				return nil, fmt.Errorf("BeforeCreateHook failed for item index [%d]: %w", i, err)
			}
		}
	} else {
		log.Warn(ctx, "No BeforeCreateHook found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Validate the types before they are added
	errs := errutil.NewMultiErr()
	for i := range elems {
		err := validate.Struct(elems[i])
		if err != nil {
			errs.AddNew("Validation failed for item at position [%d]: %w", i+1, err)
		}
	}
	if !errs.IsNil() {
		return nil, errs
	}

	// Get Columns
	var cols []string = meta.GetDatabaseColumns()

	// Get Values
	var vals [][]interface{}
	for i := range elems {
		var v = meta.GetDirectDBValues(elems[i])
		vals = append(vals, v)
	}

	// Insert Main Type
	// Construct the query
	subReq := db.InsertBuilderRequest{
		TableName:   params.TableName,
		ColumnNames: cols,
		Values:      vals,
	}

	query, args, err := conn.ConstructInsertQuery(ctx, subReq)
	if err != nil {
		return nil, fmt.Errorf("failed to construct insert query: %w", err)
	}

	rowsAffected, err := conn.ExecuteQuery(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if int(rowsAffected) != len(vals) {
		return nil, fmt.Errorf("unexpected number of rows affected, expected %d row but got %d rows", len(vals), rowsAffected)
	}

	// Insert Sub Table Fields
	for i := range elems {
		elems[i], err = meta.AddSubTableFieldsToDB(ctx, conn, params, elems[i])
		if err != nil {
			return nil, fmt.Errorf("Adding sub-table single fields to DB: %w", err)
		}
	}

	return elems, nil

}

// ListTypeByIDs fetches a list of type T based on the IDs provided.
func ListTypeByIDs[T types.BasicType, F types.Field](ctx context.Context, conn db.Connection, params db.ListTypeByIDsParams, meta ITypeDALMeta[T, F]) (ListTypeResponse[T], error) {
	var resp ListTypeResponse[T]

	// If no ID's provided, can't fetch nothing
	if len(params.IDs) < 1 {
		return resp, nil
	}

	subReq := db.SelectByIDBuilderRequest{
		TableName: params.TableName,
		Columns:   meta.GetDatabaseColumns(),
		IDColumn:  params.IDColumn,
		IDs:       params.IDs,
	}

	query, args, err := conn.ConstructSelectByIDQuery(ctx, subReq)
	if err != nil {
		return resp, err
	}

	// Run the query
	rows, err := conn.QueryRows(ctx, query, args...)
	if err != nil {
		return resp, err
	}

	// Convert rows to the item
	var elems = []T{}
	for rows.Next() {
		elem, errInner := meta.ScanDBNextRow(rows)
		if errInner != nil {
			return resp, err
		}
		elems = append(elems, elem)
	}

	llog.Debug(ctx, "Sql rows fetched", "type", meta.GetTypeCommonMeta().Name, "count", len(elems), "data", elems)

	// Unique Primary IDs of the fetched type
	var ids []scalars.ID
	for _, elem := range elems {
		ids = append(ids, elem.GetID())
	}

	// Nested Fields
	elems, err = meta.FetchSubTableFields(ctx, conn, params, elems)
	if err != nil {
		return resp, err
	}

	resp.Items = elems

	return resp, nil
}

func UpdateType[T types.BasicType, F types.Field](ctx context.Context, conn db.Connection, req UpdateTypeRequest[T, F], meta ITypeDALMeta[T, F]) (UpdateTypeResponse[T], error) {
	var resp UpdateTypeResponse[T]
	now := scalars.NewTime(time.Now().UTC())

	if req.Object.GetID().IsEmpty() {
		return resp, fmt.Errorf("object has an empty ID")
	}

	elem := req.Object

	fields := req.Fields
	excludeFields := req.ExcludeFields

	// Validation Errors: validate the Request
	var errs = errutil.NewMultiErr()

	// Validate that the ID exists
	subParams := db.ListTypeByIDsParams{
		TableName: req.TableName,
		IDColumn:  "id",
		IDs:       []scalars.ID{req.Object.GetID()},
	}
	existingElemsResp, err := ListTypeByIDs[T, F](ctx, conn, subParams, meta)
	if err != nil {
		return resp, fmt.Errorf("could not list by ID %s: %w", req.Object.GetID(), err)
	}
	if len(existingElemsResp.Items) < 1 {
		errs.AddNew("Type with ID %s does not exist: %w", req.Object.GetID(), errutil.ErrNotFound)
	}
	panics.If(len(existingElemsResp.Items) > 1, "Multiple elements founds for ID %s in table %s", req.Object.GetID(), req.TableName)
	existingElem := existingElemsResp.Items[0]

	// (Included) Fields (provided by the caller) should not include any field updatable only by DAL
	for _, f := range meta.GetCommonDALMeta().SetInternallyByDALFields {
		if types.IsFieldInFields(f, fields) {
			errs.AddNew("Mutations on field '%s' are allowed only in DAL", f)
		}
	}
	// (Included) Fields (provided by the caller) should not include any non-mutable
	for _, f := range meta.GetCommonDALMeta().ImmutableFields {
		if types.IsFieldInFields(f, fields) {
			errs.AddNew("Mutations on field '%s' are not allowed", f)
		}
	}
	if !errs.IsNil() {
		return resp, errs
	}

	// Now that we have verified that caller provided fields are okay, we can add stuff to them to make them more useful
	// - Add IsImmutable fields to ExcludeFields list
	for _, f := range meta.GetCommonDALMeta().ImmutableFields {
		if !types.IsFieldInFields(f, excludeFields) {
			excludeFields = append(excludeFields, f)
		}
	}
	// - Add DALOnlyMutable fields to ExcludeFields list
	for _, f := range meta.GetCommonDALMeta().SetInternallyByDALFields {
		if !types.IsFieldInFields(f, excludeFields) {
			excludeFields = append(excludeFields, f)
		}
	}

	// Get Updatable Columns (taking field mask into account)
	var cols = meta.GetCommonDALMeta().DatabaseColumnFields
	allowedCols := types.PruneFields(cols, fields, excludeFields)
	if len(cols) < 1 {
		return resp, fmt.Errorf("no fields provided for update: nothing to update")
	}

	llog.Debug(ctx, "Updating fields", "type", meta.GetTypeCommonMeta().Name, "columns", allowedCols)

	// Convert timestamps to UTC
	elem = meta.ConvertTimestampColumnsToUTC(elem)

	// Get changed fields and values
	colsWithValueChange, vals := meta.GetChangedFieldsAndValues(existingElem, elem, allowedCols)

	// If nothing needs to be updated, return
	if len(colsWithValueChange) < 1 {
		return resp, fmt.Errorf("No difference found in existing and provided type: %w", errutil.ErrNothingToUpdate)
	}

	// Set UpdateAt field values
	elem.SetUpdatedAt(now)

	// Add UpdatedAt fields to update columns
	colsWithValueChange = append(colsWithValueChange, meta.GetCommonDALMeta().UpdatedAtField)
	vals = append(vals, elem.GetUpdatedAt())

	// Get Query
	updateBuilderRequest := db.UpdateBuilderRequest{
		TableName: req.TableName,
		ID:        elem.GetID(),
		Columns:   FieldsToStrings(colsWithValueChange),
		Values:    vals,
	}
	query, args, err := conn.ConstructUpdateQuery(ctx, updateBuilderRequest)
	if err != nil {
		return resp, fmt.Errorf("Could not construct Update SQL query: %w", err)
	}

	rowsAffected, err := conn.ExecuteQuery(ctx, query, args...)
	if err != nil {
		return resp, err
	}
	if rowsAffected != 1 {
		return resp, fmt.Errorf("expected 1 row to be affected but got %d rows affected", rowsAffected)
	}

	// Update Nested (1:1 & 1:Many)
	elem, err = meta.UpdateSubTableFields(ctx, conn, req, allowedCols, elem)
	if err != nil {
		return resp, fmt.Errorf("Updating sub table fields: %w", err)
	}

	resp.Object = elem

	return resp, nil

}

func GetUUIDsUnion(lists ...[]scalars.ID) []scalars.ID {
	// Edge case
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}

	uniqueIDs := map[scalars.ID]bool{}

	for _, list := range lists {
		for _, id := range list {
			uniqueIDs[id] = true
		}
	}

	// See what IDs occurred in all the lists
	var common []scalars.ID
	for id := range uniqueIDs {
		common = append(common, id)
	}

	return common
}

func GetUUIDsIntersection(lists ...[]scalars.ID) []scalars.ID {
	// Edge case
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}

	// Add all ids to this map and keep a counter of how many times they occur
	// if the counter is same as number of lists, that means that ID was in all the lists
	idCount := map[scalars.ID]int{}
	for _, list := range lists {
		// Make a unique list to remove duplicates
		uniqueList := map[scalars.ID]bool{}
		for _, id := range list {
			uniqueList[id] = true
		}
		// Add the ID to idCount
		for id := range uniqueList {
			idCount[id]++
		}
	}

	// See what IDs occurred in all the lists
	var common []scalars.ID
	for id, cnt := range idCount {
		if cnt == len(lists) {
			common = append(common, id)
		}
	}

	return common
}
