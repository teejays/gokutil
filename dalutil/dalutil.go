package dalutil

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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

type IEntityDALMeta[T types.EntityTypeMutable, F types.Field] interface {
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
	AddSubTableFieldsToDB(context.Context, *db.Connection, db.InsertTypeParams, T) (T, error)
	// ForEachSubType(context.Context, func(ITypeDALMeta[T, F])) error
	// ListType
	ScanDBNextRow(context.Context, *sql.Rows) (T, error)
	FetchSubTableFields(context.Context, *db.Connection, db.ListTypeByIDsParams, []T) ([]T, error)
	// Update Type
	GetChangedFieldsAndValues(old, new T, allowedFields []F) ([]F, []interface{})
	UpdateSubTableFields(context.Context, *db.Connection, UpdateTypeRequest[T, F], []F, T, T) (T, error) // TODO

	InternalHookSavePre(ctx context.Context, elem T, now scalars.Timestamp) (T, error)
	// InternalHookCreatePre(ctx context.Context, elem T, now scalars.Timestamp) (T, error)
	// InternalHookFetchPre(ctx context.Context, elem T) (T, error)
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
	QueryText string `json:"queryText"`
}

type QueryByTextTypeRequest[T types.BasicType] struct {
	QueryText string `json:"queryText"`
}

type UpdateEntityRequest[T types.BasicType, F types.Field] struct {
	Object        T   `json:"object"`
	Fields        []F `json:"fields"`
	ExcludeFields []F `json:"excludeFields"`
}

type UpdateEntityResponse[T types.BasicType] struct {
	Object T
}

func FieldsToStrings[T types.Field](fields []T) []string {
	var colNames []string
	for _, f := range fields {
		colNames = append(colNames, f.Name().FormatSQL())
	}
	return colNames
}

func AddType[T types.BasicType, F types.Field](ctx context.Context, conn *db.Connection, params db.InsertTypeParams, meta ITypeDALMeta[T, F], elem T) (T, error) {
	log.Info(ctx, "Adding type", "type", meta.GetTypeCommonMeta().Name, "data", elem)
	var emptyT T
	items, err := BatchAddType(ctx, conn, params, meta, elem)
	if err != nil {
		return emptyT, err
	}
	if len(items) != 1 {
		return emptyT, fmt.Errorf("expected 1 item to be added but got %d items", len(items))
	}
	return items[0], nil
}

func BatchAddType[T types.BasicType, F types.Field](ctx context.Context, conn *db.Connection, params db.InsertTypeParams, meta ITypeDALMeta[T, F], elems ...T) ([]T, error) {
	now := scalars.NewTimestampNow()

	var err error

	// Make any internal changes to the data before saving and after custom hooks
	for i := range elems {
		elems[i], err = meta.InternalHookSavePre(ctx, elems[i], now)
		if err != nil {
			return nil, fmt.Errorf("Running InternalHookSavePre [item %d]: %w", i+1, err)
		}
	}

	// Add any default values for missing fields
	// Todo: this should be a part of InternalHookCreatePre
	for i := range elems {
		elems[i] = meta.SetDefaultFieldValues(elems[i])
	}

	// Run any before create hooks
	if fn := meta.GetHookCreatePre(); fn != nil {
		log.Info(ctx, "Running HookCreatePre", "type", meta.GetTypeCommonMeta().Name)
		for i := range elems {
			var err error
			elems[i], err = fn(ctx, elems[i])
			if err != nil {
				if len(elems) == 1 {
					return nil, fmt.Errorf("HookCreatePre failed: %w", err)
				} else {
					return nil, fmt.Errorf("HookCreatePre failed for item index [%d]: %w", i, err)
				}
			}
		}
	} else {
		log.Debug(ctx, "No HookCreatePre found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Run any before save hooks
	if fn := meta.GetHookSavePre(); fn != nil {
		log.Info(ctx, "Running HookSavePre", "type", meta.GetTypeCommonMeta().Name)
		for i := range elems {
			var err error
			elems[i], err = fn(ctx, elems[i])
			if err != nil {
				if len(elems) == 1 {
					return nil, fmt.Errorf("HookSavePre failed: %w", err)
				} else {
					return nil, fmt.Errorf("HookSavePre failed for item index [%d]: %w", i, err)
				}
			}
		}
	} else {
		log.Debug(ctx, "No HookSavePre found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Validate the types before they are added
	errs := errutil.NewMultiErr()
	for i := range elems {
		err := validate.Struct(elems[i])
		if err != nil {
			if len(elems) == 1 {
				errs.Wrap(err, "Validation failed")
			} else {
				errs.Wrap(err, "Validation failed for item at position [%d]", i+1)
			}
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

	query, args, err := db.ConstructInsertQuery(ctx, conn.Dialect, subReq)
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
func ListTypeByIDs[T types.BasicType, F types.Field](ctx context.Context, conn *db.Connection, params db.ListTypeByIDsParams, meta ITypeDALMeta[T, F]) (ListTypeResponse[T], error) {
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

	query, args, err := db.ConstructSelectByIDQuery(ctx, conn.Dialect, subReq)
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
		elem, errInner := meta.ScanDBNextRow(ctx, rows)
		if errInner != nil {
			return resp, errInner
		}
		elems = append(elems, elem)
	}

	llog.Debug(ctx, "SQL query executed. Rows fetched.", "type", meta.GetTypeCommonMeta().Name, "count", len(elems), "data", elems)

	// Nested Fields
	llog.Debug(ctx, "Fetching sub-table fields", "type", meta.GetTypeCommonMeta().Name)
	elems, err = meta.FetchSubTableFields(ctx, conn, params, elems)
	if err != nil {
		return resp, err
	}

	// Run any before save hooks
	if fn := meta.GetHookReadPost(); fn != nil {
		log.Info(ctx, "Running HookReadPost", "type", meta.GetTypeCommonMeta().Name)
		for i := range elems {
			var err error
			elems[i], err = fn(ctx, elems[i])
			if err != nil {
				if len(elems) == 1 {
					return resp, fmt.Errorf("HookReadPost failed: %w", err)
				} else {
					return resp, fmt.Errorf("HookReadPost failed for item index [%d]: %w", i, err)
				}
			}
		}
	} else {
		log.None(ctx, "No HookReadPost found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	resp.Items = elems

	return resp, nil
}

type UpdateTypeParams[T types.BasicType, F types.Field] struct {
	UpdateEntityRequest[T, F]
	TableName string
}

type UpdateTypeRequest[T types.BasicType, F types.Field] struct {
	Connection    *db.Connection
	TableName     string
	Object        T
	Fields        []F
	ExcludeFields []F
	Meta          ITypeDALMeta[T, F]
	AdminMode     bool
}

type UpdateTypeResponse[T types.BasicType] struct {
	Object T
}

func UpdateType[T types.BasicType, F types.Field](ctx context.Context, req UpdateTypeRequest[T, F]) (UpdateTypeResponse[T], error) {
	conn := req.Connection
	meta := req.Meta

	llog.Info(ctx, "Updating type", "type", meta.GetTypeCommonMeta().Name, "data", req.Object)
	var resp UpdateTypeResponse[T]
	now := scalars.NewTime(time.Now().UTC())

	elem := req.Object

	if elem.GetID().IsEmpty() {
		llog.Warn(ctx, "Object has an empty ID, therefore it will be added", "type", meta.GetTypeCommonMeta().Name)
		addedElem, err := AddType(ctx, conn, db.InsertTypeParams{TableName: req.TableName}, meta, req.Object)
		if err != nil {
			return resp, fmt.Errorf("could not add object with empty ID: %w", err)
		}
		resp.Object = addedElem
		return resp, nil
	}

	fields := req.Fields
	excludeFields := req.ExcludeFields

	// Validation Errors: validate the Request
	var errs = errutil.NewMultiErr()

	// Validate that the ID exists
	llog.Debug(ctx, "Fetching existing type", "type", meta.GetTypeCommonMeta().Name, "id", elem.GetID())
	subParams := db.ListTypeByIDsParams{
		TableName: req.TableName,
		IDColumn:  "id",
		IDs:       []scalars.ID{elem.GetID()},
	}
	oldElemsResp, err := ListTypeByIDs[T, F](ctx, conn, subParams, meta)
	if err != nil {
		return resp, fmt.Errorf("could not list by ID %s: %w", elem.GetID(), err)
	}
	if len(oldElemsResp.Items) < 1 {
		return resp, fmt.Errorf("Type [%s] with ID [%s] not found: %w", meta.GetTypeCommonMeta().Name, elem.GetID(), errutil.ErrNotFound)
	}
	panics.If(len(oldElemsResp.Items) > 1, "Multiple elements founds for ID %s in table %s", elem.GetID(), req.TableName)
	oldElem := oldElemsResp.Items[0]

	// (Included) Fields (provided by the caller) should not include any field updatable only by DAL
	if !req.AdminMode {
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
	}
	if !errs.IsNil() {
		return resp, errs
	}

	// Now that we have verified that caller provided fields are okay, we can add stuff to them to make them more useful
	if !req.AdminMode {
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
	}

	// Get updatable fields (taking field mask into account)
	allFields := meta.GetTypeCommonMeta().Fields
	allowedFields := types.PruneFields(allFields, fields, excludeFields)
	if len(allowedFields) < 1 {
		return resp, fmt.Errorf("no fields available for update")
	}

	// Make any internal changes to the data before saving and before custom hooks
	elem, err = meta.InternalHookSavePre(ctx, elem, now)
	if err != nil {
		return resp, fmt.Errorf("Running InternalHookSavePre (before custom hooks): %w", err)
	}

	// Update Pre Hooks
	if fn := meta.GetHookUpdatePre(); fn != nil {
		log.Info(ctx, "Running HookUpdatePre", "type", meta.GetTypeCommonMeta().Name)
		var err error
		elem, err = fn(ctx, elem)
		if err != nil {
			return resp, fmt.Errorf("HookUpdatePre failed: %w", err)
		}

	} else {
		log.None(ctx, "No HookUpdatePre found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Save Pre Hooks
	if fn := meta.GetHookSavePre(); fn != nil {
		log.Info(ctx, "Running HookSavePre", "type", meta.GetTypeCommonMeta().Name)
		var err error
		elem, err = fn(ctx, elem)
		if err != nil {
			return resp, fmt.Errorf("HookSavePre failed: %w", err)
		}

	} else {
		log.None(ctx, "No HookSavePre found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Direct table
	{
		cols := meta.GetCommonDALMeta().DatabaseColumnFields
		allowedCols := types.PruneFields(allowedFields, cols, nil)
		if len(allowedCols) < 1 {
			// nothing to update on the main table
			llog.Warn(ctx, "No direct database columns to update", "type", meta.GetTypeCommonMeta().Name)
		} else {

			llog.Debug(ctx, "Updating direct database fields", "type", meta.GetTypeCommonMeta().Name, "columns", allowedCols)

			// Get changed fields and values
			colsWithValueChange, vals := meta.GetChangedFieldsAndValues(oldElem, elem, allowedCols)
			// If nothing needs to be updated, return
			if len(colsWithValueChange) < 1 {
				llog.Warn(ctx, "Updating Type: No difference found in existing and new type, nothing to update.", "type", meta.GetTypeCommonMeta().Name)
			} else {

				// Add UpdatedAt fields to update columns
				colsWithValueChange = append(colsWithValueChange, meta.GetCommonDALMeta().UpdatedAtField)
				vals = append(vals, elem.GetUpdatedAt())

				// Get Query
				updateBuilderRequest := db.UpdateBuilderRequest{
					TableName:        req.TableName,
					IdentifierColumn: "id",
					IdentifierValue:  elem.GetID(),
					Columns:          FieldsToStrings(colsWithValueChange),
					Values:           vals,
				}
				query, args, err := db.ConstructUpdateQuery(ctx, conn.Dialect, updateBuilderRequest)
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
			}
		}
	}

	// Update Nested (1:1 & 1:Many)
	elem, err = meta.UpdateSubTableFields(ctx, conn, req, allowedFields, elem, oldElem)
	if err != nil {
		return resp, fmt.Errorf("Updating sub table fields: %w", err)
	}

	// Run any after save hooks
	// Save Post Hooks
	if fn := meta.GetHookSavePost(); fn != nil {
		log.Info(ctx, "Running HookSavePost", "type", meta.GetTypeCommonMeta().Name)
		var err error
		elem, err = fn(ctx, elem)
		if err != nil {
			return resp, fmt.Errorf("HookSavePost failed: %w", err)
		}

	} else {
		log.None(ctx, "No HookSavePost found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
	}

	// Update Post Hooks
	if fn := meta.GetHookUpdatePost(); fn != nil {
		log.Info(ctx, "Running HookUpdatePost", "type", meta.GetTypeCommonMeta().Name)
		var err error
		elem, err = fn(ctx, elem)
		if err != nil {
			return resp, fmt.Errorf("HookUpdatePost failed: %w", err)
		}
	} else {
		log.None(ctx, "No HookUpdatePost found", "type", meta.GetTypeCommonMeta().Name)
	}

	resp.Object = elem

	return resp, nil

}

/* * * * * * *
 * Delete
 * * * * * * */

type DeleteEntityRequest struct {
	ID scalars.ID
}

type DeleteTypeRequest[T types.BasicType, F types.Field] struct {
	Connection *db.Connection
	TableName  string
	ObjectID   scalars.ID
	Meta       ITypeDALMeta[T, F]
	Now        scalars.Timestamp
}

type DeleteTypeResponse struct {
	ObjectID  scalars.ID        `json:"objectId" yaml:"objectId"`
	DeletedAt scalars.Timestamp `json:"deletedAt" yaml:"deletedAt"`
}

func DeleteType[T types.BasicType, F types.Field](ctx context.Context, req DeleteTypeRequest[T, F]) (DeleteTypeResponse, error) {
	panics.IfNil(req.Connection, "dalutil.DeleteType() called with nil Connection")
	panics.If(req.Now.IsEmpty(), "dalutil.DeleteType() called with empty timestamp")

	var resp = DeleteTypeResponse{
		ObjectID:  req.ObjectID,
		DeletedAt: req.Now,
	}
	typName := req.Meta.GetTypeCommonMeta().Name

	llog.Info(ctx, "Deleting type", "type", typName, "id", req.ObjectID)

	// elem := req.Object
	if req.ObjectID.IsEmpty() {
		return resp, fmt.Errorf("ID is empty, cannot delete")
	}

	// Get the existing element.
	llog.Debug(ctx, "Fetching existing type", "type", typName, "id", req.ObjectID)
	subParams := db.ListTypeByIDsParams{
		TableName: req.TableName,
		IDColumn:  "id",
		IDs:       []scalars.ID{req.ObjectID},
	}
	oldElemsResp, err := ListTypeByIDs[T, F](ctx, req.Connection, subParams, req.Meta)
	if err != nil {
		return resp, fmt.Errorf("could not list by ID %s: %w", req.ObjectID, err)
	}
	if len(oldElemsResp.Items) < 1 {
		return resp, errutil.NewGerror("Type [%s] with ID [%s] not found", typName, req.ObjectID).
			SetHTTPStatus(http.StatusNotFound).
			SetExternalMsg("We could not find any entity or type with the given ID.")
	}
	panics.If(len(oldElemsResp.Items) > 1, "Multiple elements founds for ID %s in table %s", req.ObjectID, req.TableName)
	oldElem := oldElemsResp.Items[0]
	if oldElem.GetDeletedAt() != nil {
		return resp, errutil.NewGerror("Type [%s] with ID [%s] is already deleted", typName, req.ObjectID).
			SetHTTPStatus(http.StatusConflict).
			SetExternalMsg("Entity or type is already marked as deleted.")
	}

	// Run any before delete hooks: Not implemented
	if fn := req.Meta.GetHookDeletePre(); fn != nil {
		log.Info(ctx, "Running HookDeletePre", "type", typName)
		return resp, fmt.Errorf("HookDeletePre not implemented")
	}

	// Run the update query
	{
		columns := []string{"deleted_at"}
		llog.Debug(ctx, "Updating direct database fields", "type", typName, "columns", columns)

		// Get Query
		updateBuilderRequest := db.UpdateBuilderRequest{
			TableName:        req.TableName,
			IdentifierColumn: "id",
			IdentifierValue:  req.ObjectID,
			Columns:          columns,
			Values:           []interface{}{req.Now},
		}
		query, args, err := db.ConstructUpdateQuery(ctx, req.Connection.Dialect, updateBuilderRequest)
		if err != nil {
			return resp, fmt.Errorf("Could not construct Update SQL query: %w", err)
		}

		rowsAffected, err := req.Connection.ExecuteQuery(ctx, query, args...)
		if err != nil {
			return resp, err
		}
		if rowsAffected != 1 {
			return resp, fmt.Errorf("expected 1 row to be affected but got %d rows affected", rowsAffected)
		}

	}

	// TODO:  Delete any sub-table fields

	// Run any after delete hooks: Not implemented
	if fn := req.Meta.GetHookDeletePost(); fn != nil {
		log.Info(ctx, "Running HookDeletePost", "type", typName)
		return resp, fmt.Errorf("HookDeletePost not implemented")
	}

	return resp, nil

}

// type SaveBatchTypeParams[T types.BasicType, F types.Field] struct {
// 	TableName  string
// 	Connection *db.Connection
// 	Objects    []T
// 	ObjectMeta ITypeDALMeta[T, F]
// 	Now        scalars.Timestamp
// }

// func SaveBatchType[T types.BasicType, F types.Field](ctx context.Context, p SaveBatchTypeParams[T, F]) error {

// 	var err error

// 	// Populate vars from params for easier access
// 	elems := p.Objects
// 	meta := p.ObjectMeta
// 	now := p.Now
// 	if now.IsEmpty() {
// 		now = scalars.NewTimestampNow()
// 	}

// 	// Standard internal pre-save hook: populate ID, time fields etc.
// 	for i := range elems {
// 		elems[i], err = meta.InternalHookSavePre(ctx, elems[i], now)
// 		if err != nil {
// 			return errutil.Wrap(err, "InternalHook [SavePre] [item %d]", i+1)
// 		}
// 	}

// 	// A few things we need to do only for new items.
// 	for i := range elems {

// 		isNew := elems[i].GetID().IsEmpty()
// 		if !isNew {
// 			continue
// 		}

// 		// For new items, we should add any default value for missing fields (when specified)
// 		// Todo: this should be a part of InternalHookCreatePre?
// 		elems[i] = meta.SetDefaultFieldValues(elems[i])

// 		// Run any before create hooks
// 		if fn := meta.GetHookCreatePre(); fn != nil {
// 			log.Info(ctx, "SaveType: Running Hook [CreatePre]", "type", meta.GetTypeCommonMeta().Name)
// 			elems[i], err = fn(ctx, elems[i])
// 			if err != nil {
// 				if len(elems) == 1 {
// 					return errutil.Wrap(err, "Hook failed [CreatePre]")
// 				} else {
// 					return errutil.Wrap(err, "Hook failed [CreatePre] [item %d]", i+1)
// 				}
// 			}
// 		} else {
// 			log.Debug(ctx, "SaveType: No HookCreatePre found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
// 		}

// 	}

// 	// New items: Create Pre Hooks
// 	if fn := meta.GetHookCreatePre(); fn != nil {
// 		log.Info(ctx, "SaveType: Running Hook [CreatePre]", "type", meta.GetTypeCommonMeta().Name)

// 		for i := range elems {
// 			isNew := elems[i].GetID().IsEmpty()
// 			if !isNew {
// 				continue
// 			}

// 			elems[i], err = fn(ctx, elems[i])
// 			if err != nil {
// 				if len(elems) == 1 {
// 					return errutil.Wrap(err, "Hook failed [CreatePre]")
// 				} else {
// 					return errutil.Wrap(err, "Hook failed [CreatePre] [item %d]", i+1)
// 				}
// 			}
// 		}
// 	} else {
// 		log.Debug(ctx, "SaveType: No Hook [CreatePre] found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
// 	}

// 	// All items: Save Pre Hooks
// 	if fn := meta.GetHookSavePre(); fn != nil {
// 		log.Info(ctx, "SaveType: Running Hook [SavePre]", "type", meta.GetTypeCommonMeta().Name)
// 		for i := range elems {
// 			var err error
// 			elems[i], err = fn(ctx, elems[i])
// 			if err != nil {
// 				if len(elems) == 1 {
// 					return errutil.Wrap(err, "Hook failed [SavePre]")
// 				} else {
// 					return errutil.Wrap(err, "Hook failed [SavePre] [item %d]", i+1)
// 				}
// 			}
// 		}
// 	} else {
// 		log.Debug(ctx, "SaveType: No Hook [SavePre] found", "type", meta.GetTypeCommonMeta().Name, "meta", meta)
// 	}

// 	// Validate the types before they are added
// 	// Note: We've only mutated/hooked the outer type so we can really only validate that.
// 	errs := errutil.NewMultiErr()
// 	for i := range elems {
// 		err := validate.Struct(elems[i])
// 		if err != nil {
// 			if len(elems) == 1 {
// 				errs.Wrap(err, "Validation failed")
// 			} else {
// 				errs.Wrap(err, "Validation failed [item %d]", i+1)
// 			}
// 		}
// 	}
// 	if !errs.IsNil() {
// 		return errs
// 	}

// 	// Get Columns
// 	var cols []string = meta.GetDatabaseColumns()

// 	// // Get Values (each value is a row/elem)
// 	// var vals [][]interface{}
// 	// for i := range elems {
// 	// 	var v = meta.GetDirectDBValues(elems[i])
// 	// 	vals = append(vals, v)
// 	// }

// 	// Insert Main Type

// 	// Our upsert helper can only handle one upsert query generation at time (because it individually checks if we need to insert or update)
// 	// So we will have to do this in a loop
// 	for i := range elems {

// 		vals := meta.GetDirectDBValues(elems[i])

// 		// Construct the query
// 		subReq := db.ConstructUpsertQueryRequest{
// 			Dialect:      p.Connection.Dialect,
// 			TableName:    p.TableName,
// 			UpsertColumn: "id",
// 			ColumnNames:  cols,
// 			Values:       vals,
// 		}

// 		query, args, err := db.ConstructUpsertQuery(ctx, subReq)
// 		if err != nil {
// 			return errutil.Wrap(err, "constructing upsert query")
// 		}

// 		rowsAffected, err := p.Connection.ExecuteQuery(ctx, query, args...)
// 		if err != nil {
// 			return errutil.Wrap(err, "executing query")
// 		}

// 		if int(rowsAffected) != len(vals) {
// 			return fmt.Errorf("unexpected number of rows affected, expected %d row but got %d rows", len(vals), rowsAffected)
// 		}

// 		// Enter the tree (sub-types etc.)

// 		err = meta.ForEachSubType(ctx,
// 			func(subMeta ITypeDALMeta) {
// 				// do something
// 			},
// 		)

// 	}

// 	return nil

// }

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
