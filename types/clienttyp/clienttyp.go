package clienttyp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/teejays/gokutil/dalutil"
	"github.com/teejays/gokutil/httputil"
	"github.com/teejays/gokutil/types"
)

type EntityBaseClient[T types.EntityType, inT any, F types.Field, Flt types.FilterType] interface {
	Add(ctx context.Context, req dalutil.EntityAddRequest[inT]) (T, error)
	Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error)
	Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error)
	List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error)
	QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error)
}

// entityHTTPBaseClient is a HTTP protocol implementation of the EntityClientI
type entityHTTPBaseClient[T types.BasicType, inT any, F types.Field, Flt types.FilterType] struct {
	client  http.Client
	baseURL string
}

func NewEntityHTTPClient[T types.BasicType, inT any, F types.Field, Flt types.FilterType](ctx context.Context, baseURL string) (EntityBaseClient[T, inT, F, Flt], error) {
	return entityHTTPBaseClient[T, inT, F, Flt]{
		client:  http.Client{},
		baseURL: baseURL,
	}, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Add(ctx context.Context, req dalutil.EntityAddRequest[inT]) (T, error) {
	resp, err := httputil.MakeRequest[dalutil.EntityAddRequest[inT], T](ctx, c.client, http.MethodPost, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPost, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.UpdateEntityRequest[T, F], dalutil.UpdateEntityResponse[T]](ctx, c.client, http.MethodPut, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPut, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error) {
	resp, err := httputil.MakeRequest[dalutil.GetEntityRequest[T], T](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.ListEntityRequest[Flt], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.QueryByTextEntityRequest[T], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}
