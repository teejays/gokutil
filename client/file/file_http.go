package client_file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/teejays/gokutil/gopi"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/scalars"
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

func (c HTTPFileClient) FileDataUpload(ctx context.Context, data []byte) (scalars.ID, error) {
	return c.FileUpload(ctx, bytes.NewReader(data))
}

// FileRead reads a file from the local file system.
func (c HTTPFileClient) FileRead(ctx context.Context, id scalars.ID) (io.ReadCloser, error) {

	// Make a HTTP GET request to the server to read the file.
	return nil, fmt.Errorf("not implemented")
}

func (c HTTPFileClient) FileDataRead(ctx context.Context, id scalars.ID) ([]byte, error) {
	// Make a HTTP GET request to the server to read the file.
	return nil, fmt.Errorf("not implemented")
}
