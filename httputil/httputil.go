package httputil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/teejays/clog"
	"github.com/teejays/goku-util/validate"
)

// ErrEmptyBody is used when we expect to receive a request with some body but we don't
var ErrEmptyBody = fmt.Errorf("no content provided with the HTTP request")

// ErrInvalidJSON is used when we expect to receive a JSON request but we don't
var ErrInvalidJSON = fmt.Errorf("content is not a valid JSON")

func UnmarshalJSONFromRequest(r *http.Request, v interface{}) error {
	// Read the HTTP request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// api.WriteError(w, http.StatusBadRequest, err, false, nil)
		return err
	}
	defer r.Body.Close()

	if len(body) < 1 {
		// api.WriteError(w, http.StatusBadRequest, api.ErrEmptyBody, false, nil)
		return ErrEmptyBody
	}

	clog.Debugf("api: Unmarshaling to JSON: body:\n%+v", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, &v)
	if err != nil {
		// api.WriteError(w, http.StatusBadRequest, err, true, api.ErrInvalidJSON)
		clog.Errorf("api: Error unmarshaling JSON: %v", err)
		return ErrInvalidJSON
	}

	err = validate.Struct(v)
	if err != nil {
		return err
	}

	return nil
}
