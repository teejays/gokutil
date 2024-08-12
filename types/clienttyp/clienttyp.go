package clienttyp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/teejays/gokutil/dalutil"
	"github.com/teejays/gokutil/httputil"
	"github.com/teejays/gokutil/types"
)

type EntityBaseClient[T types.EntityType, F types.Field, Flt types.FilterType] interface {
	Add(ctx context.Context, req T) (T, error)
	Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error)
	Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error)
	List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error)
	QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error)
}

// type ServiceBaseClient interface {
// 	GetBaseClient() ServiceBaseClientImpl
// }

// type ServiceBaseClientImpl struct {
// 	Connection db.Connection
// }

// // EntityDALClient is a database protocol implementation of the EntityClientI
// type EntityDALClient[T types.BasicType, F types.Field] struct {
// 	conn db.Connection
// }

// func NewEntityDALClient[T types.BasicType, F types.Field](conn db.Connection) EntityDALClient[T, F] {
// 	return EntityDALClient[T, F]{
// 		conn: conn,
// 	}
// }

// entityHTTPBaseClient is a HTTP protocol implementation of the EntityClientI
type entityHTTPBaseClient[T types.BasicType, F types.Field, Flt types.FilterType] struct {
	client  http.Client
	baseURL string
}

func NewEntityHTTPClient[T types.BasicType, F types.Field, Flt types.FilterType](ctx context.Context, baseURL string) (EntityBaseClient[T, F, Flt], error) {
	return entityHTTPBaseClient[T, F, Flt]{
		client:  http.Client{},
		baseURL: baseURL,
	}, nil
}

func (c entityHTTPBaseClient[T, F, Flt]) Add(ctx context.Context, req T) (T, error) {
	resp, err := httputil.MakeRequest[T, T](ctx, c.client, http.MethodPost, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPost, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, F, Flt]) Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.UpdateEntityRequest[T, F], dalutil.UpdateEntityResponse[T]](ctx, c.client, http.MethodPut, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPut, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, F, Flt]) Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error) {
	resp, err := httputil.MakeRequest[dalutil.GetEntityRequest[T], T](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, F, Flt]) List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.ListEntityRequest[Flt], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, F, Flt]) QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.QueryByTextEntityRequest[T], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}
