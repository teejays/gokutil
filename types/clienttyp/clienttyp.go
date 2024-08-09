package clienttyp

import (
	"context"

	"github.com/teejays/gokutil/client/db"
	"github.com/teejays/gokutil/dalutil"
	"github.com/teejays/gokutil/types"
)

type EntityClient[T types.EntityType] interface {
	Add(ctx context.Context, req T) (T, error)
	Get(ctx context.Context, req dalutil.GetEntityRequest[T]) (T, error)
	Update(ctx context.Context, req T) (T, error)
	List(ctx context.Context, req dalutil.ListEntityRequest[T]) (dalutil.ListEntityResponse[T], error)
	QueryByText(ctx context.Context, req dalutil.QueryByTextEntityRequest[T]) (dalutil.ListEntityResponse[T], error)
}

type ServiceBaseClient interface {
	GetBaseClient() ServiceBaseClientImpl
}

type ServiceBaseClientImpl struct {
	Connection db.Connection
}
