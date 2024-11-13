package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

func MakeRequest[ReqT any, RespT any](ctx context.Context, c http.Client, method string, req ReqT) (RespT, error) {

	var resp RespT

	// Create the request
	reqBody := new(bytes.Buffer)
	err := json.NewEncoder(reqBody).Encode(req)
	if err != nil {
		return resp, fmt.Errorf("Converting request to JSON: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "user/user", reqBody)
	if err != nil {
		return resp, fmt.Errorf("Creating HTTP request: %w", err)
	}

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
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return resp, fmt.Errorf("Parsing HTTP response: %w", err)
	}

	return resp, nil
}
