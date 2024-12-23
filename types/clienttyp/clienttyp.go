package clienttyp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/teejays/gokutil/dalutil"
	"github.com/teejays/gokutil/gopi"
	"github.com/teejays/gokutil/httputil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/types"
)

type EntityBaseClient[T types.EntityType, inT any, F types.Field, Flt types.FilterType] interface {
	Add(ctx context.Context, req dalutil.EntityAddRequest[inT]) (T, error)
	Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error)
	Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error)
	Delete(ctx context.Context, req dalutil.DeleteEntityRequest) (dalutil.DeleteTypeResponse, error)
	List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error)
	QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error)
}

// entityHTTPBaseClient is a HTTP protocol implementation of the EntityClientI
type entityHTTPBaseClient[T types.BasicType, inT any, F types.Field, Flt types.FilterType] struct {
	client  http.Client
	baseURL string
	bearer  string
}

func NewEntityHTTPClient[T types.BasicType, inT any, F types.Field, Flt types.FilterType](ctx context.Context, baseURL, bearer string) (EntityBaseClient[T, inT, F, Flt], error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if bearer == "" {
		log.Warn(ctx, "[Entity Client HTTP] No bearer auth token provided. This client will not be able to authenticate with the server")
	}
	return entityHTTPBaseClient[T, inT, F, Flt]{
		client:  http.Client{},
		baseURL: baseURL,
		bearer:  bearer,
	}, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Add(ctx context.Context, req dalutil.EntityAddRequest[inT]) (T, error) {
	resp, err := httputil.MakeRequest[dalutil.EntityAddRequest[inT], T](ctx, c.client, http.MethodPost, c.baseURL, c.bearer, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPost, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Update(ctx context.Context, req dalutil.UpdateEntityRequest[T, F]) (dalutil.UpdateEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.UpdateEntityRequest[T, F], dalutil.UpdateEntityResponse[T]](ctx, c.client, http.MethodPut, c.baseURL, c.bearer, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodPut, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error) {
	resp, err := httputil.MakeRequest[dalutil.GetEntityRequest[T], gopi.StandardResponseGeneric[T]](ctx, c.client, http.MethodGet, c.baseURL, c.bearer, req)
	if err != nil {
		return resp.Data, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	if resp.Error != "" {
		return resp.Data, fmt.Errorf("Non-successful response from the server: %s", resp.Error)
	}
	return resp.Data, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) Delete(ctx context.Context, req dalutil.DeleteEntityRequest) (dalutil.DeleteTypeResponse, error) {
	resp, err := httputil.MakeRequest[dalutil.DeleteEntityRequest, dalutil.DeleteTypeResponse](ctx, c.client, http.MethodDelete, c.baseURL, c.bearer, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodDelete, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) List(ctx context.Context, req dalutil.ListEntityRequest[Flt]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.ListEntityRequest[Flt], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, fmt.Sprintf("%s/list", c.baseURL), c.bearer, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}

func (c entityHTTPBaseClient[T, inT, F, Flt]) QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error) {
	resp, err := httputil.MakeRequest[dalutil.QueryByTextEntityRequest[T], dalutil.ListEntityResponse[T]](ctx, c.client, http.MethodGet, fmt.Sprintf("%s/query_by_text", c.baseURL), c.bearer, req)
	if err != nil {
		return resp, fmt.Errorf("Making HTTP %s request: %w", http.MethodGet, err)
	}
	return resp, nil
}
