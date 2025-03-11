package client_file

import (
	"context"
	"fmt"
	"io"

	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/scalars"
)

/* * * * * *
 * Singleton
 * * * * * */

var _defaultFileClient IFileClient

func InitDefaultFileClient(ctx context.Context, c IFileClient) error {
	if _defaultFileClient != nil {
		log.Error(ctx, "InitDefaultFileClient: defaultFileClient is already setup. Overwriting it.", "client", c)
	}
	if c == nil {
		log.Error(ctx, "InitDefaultFileClient: client provided is nil. Clearing the defaultFileClient.")
	}
	_defaultFileClient = c
	return nil
}

func GetDefaultFileClient(ctx context.Context) (IFileClient, error) {
	if _defaultFileClient == nil {
		return nil, fmt.Errorf("defaultFileClient is not initialized")
	}
	return _defaultFileClient, nil
}

/* * * * * *
 * IFileClient
 * * * * * */

// IFileClient is an interface that should be implemented by any file service provider we integrate in our app.
// e.g. LocalFileClient, S3FileClient, etc.
type IFileClient interface {
	GetClientType() ClientType
	FileUpload(context.Context, io.Reader) (scalars.ID, error)
	FileDataUpload(context.Context, []byte) (scalars.ID, error)
	FileRead(context.Context, scalars.ID) (io.ReadCloser, error)
	FileDataRead(context.Context, scalars.ID) ([]byte, error)
}

type ClientType string

const (
	TypeLocal ClientType = "local-filesystem"
	TypeHTTP  ClientType = "http"
	TypeS3    ClientType = "s3"
)

/* * * * * *
 * S3FileClient
 * * * * * */
