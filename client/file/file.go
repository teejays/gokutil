package client_file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/teejays/gokutil/gopi"
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
	TypeHTTP  ClientType = "http"
)

/* * * * * *
 * HTTPFileClient
 * * * * * */

// HTTPFileClient is a client that uploads/reads files through HTTP calls.
type HTTPFileClient struct {
	httpClient  *http.Client
	baseURL     string
	bearerToken string
}

// NewHTTPFileClient creates a new HTTPFileClient
func NewHTTPFileClient(ctx context.Context, baseURL, bearerToken string) (IFileClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if bearerToken == "" {
		log.Warn(ctx, "No bearer auth token provided. This client will not be able to authenticate with the server")
	}

	return HTTPFileClient{
		httpClient:  &http.Client{},
		baseURL:     baseURL,
		bearerToken: bearerToken,
	}, nil
}

func (c HTTPFileClient) GetClientType() ClientType {
	return TypeLocal
}

type FileUploadResponse struct {
	ID scalars.ID `json:"id"`
}

// FileUpload uploads a file to the local file system.
func (c HTTPFileClient) FileUpload(ctx context.Context, file io.Reader) (scalars.ID, error) {
	// Make a HTTP POST request to the server to upload the file.
	// The file should be sent as a multipart form data.
	// The server should return the ID of the file that was uploaded.
	var ret scalars.ID

	// Create a buffer to hold the multipart data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create the file field in the multipart form
	part, err := writer.CreateFormFile("file", "unknown-name")
	if err != nil {
		return ret, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy the file into the multipart field
	if _, err := io.Copy(part, file); err != nil {
		return ret, fmt.Errorf("failed to copy file: %w", err)
	}

	// Close the writer to finalize the multipart form
	if err := writer.Close(); err != nil {
		return ret, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", c.baseURL, &body)
	if err != nil {
		return ret, fmt.Errorf("failed to create request: %w", err)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
	}

	// Set the content type to the multipart form's content type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ret, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Debug(ctx, "File uploaded", "url", c.baseURL, "response", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return ret, fmt.Errorf("failed to upload file: %s", respBody)
	}

	// Parse the response
	var uploadResp gopi.StandardResponseGeneric[FileUploadResponse]
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return ret, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if uploadResp.Error != "" {
		return ret, fmt.Errorf("failed to upload file: %s", uploadResp.Error)
	}

	return uploadResp.Data.ID, nil

}

// FileRead reads a file from the local file system.
func (c HTTPFileClient) FileRead(ctx context.Context, id scalars.ID) (io.ReadCloser, error) {

	// Make a HTTP GET request to the server to read the file.
	return nil, fmt.Errorf("not implemented")
}

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
