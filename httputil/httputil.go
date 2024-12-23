package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	jsonutil "github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/validate"
)

var llog = log.GetLogger().WithHeading("httputil")

// ErrEmptyBody is used when we expect to receive a request with some body but we don't
var ErrEmptyBody = fmt.Errorf("no content provided with the HTTP request")

// ErrInvalidJSON is used when we expect to receive a JSON request but we don't
var ErrInvalidJSON = fmt.Errorf("content is not a valid JSON")

func UnmarshalJSONFromRequest[ReqT any](r *http.Request, v *ReqT) error {
	ctx := r.Context()

	// Read the HTTP request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if len(body) < 1 {
		return ErrEmptyBody
	}

	llog.Debug(ctx, "Unmarshaling to JSON", "body", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, v)
	if err != nil {
		llog.Error(ctx, "Couldn't unmarshal JSON", "error", err)
		return ErrInvalidJSON
	}

	llog.Warn(ctx, "Unmarshaled JSON", "v", v)

	err = validate.Struct(v)
	if err != nil {
		return fmt.Errorf("Validation error: %w", err)
	}

	return nil
}

func MakeRequest[ReqT any, RespT any](ctx context.Context, c http.Client, method string, path string, bearer string, req ReqT) (RespT, error) {

	var resp RespT

	if path == "" {
		return resp, fmt.Errorf("URL is empty")
	}
	// If URL doesn't start with http or https, add http
	if path[:4] != "http" {
		path = "http://" + path
	}

	// Create the request
	// If method is GET, we need to encode the request as query params
	// If method is POST, we need to encode the request as JSON
	var reqBody io.Reader
	switch method {
	case http.MethodGet:
		// Encode the request as query params
		parsedURL, err := url.Parse(path)
		if err != nil {
			return resp, fmt.Errorf("Parsing URL: %w", err)
		}
		q := parsedURL.Query()
		reqBytes, err := jsonutil.Marshal(req)
		if err != nil {
			return resp, fmt.Errorf("Marshaling request to JSON: %w", err)
		}
		q.Add("req", string(reqBytes))
		parsedURL.RawQuery = q.Encode()
		path = parsedURL.String()
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		// Encode the request as JSON
		buff := bytes.NewBuffer(nil)
		err := json.NewEncoder(buff).Encode(req)
		if err != nil {
			return resp, fmt.Errorf("Converting request to JSON: %w", err)
		}
		reqBody = buff
	default:
		return resp, fmt.Errorf("Unsupported HTTP method: %s", method)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, path, reqBody)
	if err != nil {
		return resp, fmt.Errorf("Creating HTTP request: %w", err)
	}

	// If the req has no headers, add a default content type
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
		log.Warn(ctx, "[HTTP] No content type provided in the request. Setting it to application/json")
	}

	// Add the bearer token if provided
	if bearer != "" {
		httpReq.Header.Set("Authorization", "Bearer "+bearer)
	}

	log.Debug(ctx, "[HTTP] Making request", "url", path, "method", method, "request", jsonutil.MustPrettyPrint(req), "headers", httpReq.Header)

	// Send the request
	httpResp, err := c.Do(httpReq)
	if err != nil {
		return resp, fmt.Errorf("Sending HTTP request: %w", err)
	}
	defer httpResp.Body.Close()

	// Parse the response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, fmt.Errorf("Reading HTTP response: %w", err)
	}

	log.Debug(ctx, "[HTTP] Response received", "url", path, "method", method, "status", httpResp.Status, "body", string(respBody))
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return resp, fmt.Errorf("Parsing HTTP response: %w", err)
	}

	return resp, nil
}
