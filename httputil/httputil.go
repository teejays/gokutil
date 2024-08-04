package httputil

import (
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

func UnmarshalJSONFromRequest(r *http.Request, v interface{}) error {
	// Read the HTTP request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if len(body) < 1 {
		return ErrEmptyBody
	}

	llog.Debug(r.Context(), "unmarshaling to JSON", "body", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, &v)
	if err != nil {
		llog.Error(r.Context(), "Couldn't unmarshaling JSON", "error", err)
		return ErrInvalidJSON
	}

	err = validate.Struct(v)
	if err != nil {
		return err
	}

	return nil
}
