package client_file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/scalars"
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

func (c LocalFileClient) FileDataUpload(ctx context.Context, data []byte) (scalars.ID, error) {
	// Convert the file to a reader and call FileUpload
	return c.FileUpload(ctx, bytes.NewReader(data))
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

func (c LocalFileClient) FileDataRead(ctx context.Context, id scalars.ID) ([]byte, error) {
	// Convert the file to a reader and call FileRead
	file, err := c.FileRead(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	defer file.Close()
	return io.ReadAll(file)
}
