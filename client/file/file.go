package client_file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	FileRead(context.Context, scalars.ID) (io.ReadCloser, error)
}

type ClientType string

const (
	TypeLocal ClientType = "local-filesystem"
)

/* * * * * *
 * LocalFileClient
 * * * * * */

// LocalFileClient is a client that interacts with the local file system. This is useful for DEV.
// It creates a dir within the rootDir, and stores all it files in that directory.
type LocalFileClient struct {
	baseDir string
}

// NewLocalFileClient creates a new LocalFileClient
func NewLocalFileClient(ctx context.Context, rootDir string) (IFileClient, error) {
	baseDir := filepath.Join(rootDir, "file-client-local")

	err := os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		return LocalFileClient{}, fmt.Errorf("creating base directory [%s]: %w", baseDir, err)
	}

	return LocalFileClient{
		baseDir: baseDir,
	}, nil
}

func (c LocalFileClient) GetClientType() ClientType {
	return TypeLocal
}

// FileUpload uploads a file to the local file system.
func (c LocalFileClient) FileUpload(ctx context.Context, r io.Reader) (scalars.ID, error) {
	// create a file in the rootDir + "/local-file-client" + clientID
	// and copy the content of the reader to the file.

	fileID := scalars.NewID()
	filePath := filepath.Join(c.baseDir, fileID.String())
	file, err := os.Create(filePath)
	if err != nil {
		return scalars.ID{}, fmt.Errorf("creating empyty file [%s]: %w", filePath, err)
	}
	defer file.Close()
	n, err := io.Copy(file, r)
	if err != nil {
		return scalars.ID{}, fmt.Errorf("copying content to file [%s]: %w", filePath, err)
	}
	if n == 0 {
		return scalars.ID{}, fmt.Errorf("no content copied to file [%s]", filePath)
	}
	log.Debug(ctx, "File uploaded", "fileID", fileID, "filePath", filePath, "bytesCopied", n)

	return fileID, nil
}

// FileRead reads a file from the local file system.
func (c LocalFileClient) FileRead(ctx context.Context, id scalars.ID) (io.ReadCloser, error) {
	filePath := filepath.Join(c.baseDir, id.String())
	// Open the dir, ensure there is only file, and read the file

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file [%s]: %w", filePath, err)
	}
	return file, nil
}
