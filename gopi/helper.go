package gopi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/httputil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"

	"github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/gopi/validator"
)

// GetQueryParamInt extracts the param value with given name out of the URL query
func GetQueryParamInt(r *http.Request, name string, defaultVal int) (int, error) {
	err := r.ParseForm()
	if err != nil {
		return defaultVal, err
	}
	values, exist := r.Form[name]
	log.DebugWithoutCtx("[GetQueryParamInt] URL values", "param", name, "value", values)

	if !exist {
		return defaultVal, nil
	}
	if len(values) > 1 {
		return defaultVal, fmt.Errorf("multiple URL form values found for %s", name)
	}

	val, err := strconv.Atoi(values[0])
	if err != nil {
		return defaultVal, fmt.Errorf("error parsing %s value to an int: %v", name, err)
	}
	return val, nil
}

// GetMuxParamInt extracts the param with given name out of the route path
func GetMuxParamInt(r *http.Request, name string) (int64, error) {

	var vars = mux.Vars(r)

	log.DebugWithoutCtx("[GetMuxParamInt] MUX vars", "value", vars)
	valStr := vars[name]
	if strings.TrimSpace(valStr) == "" {
		return -1, fmt.Errorf("could not find var %s in the route", name)
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return -1, fmt.Errorf("could not convert var %s to an int64: %v", name, err)
	}

	return int64(val), nil
}

// GetMuxParamStr extracts the param with given name out of the route path
func GetMuxParamStr(r *http.Request, name string) (string, error) {

	var vars = mux.Vars(r)
	log.DebugWithoutCtx("[GetMuxParamStr] MUX vars", "value", vars)
	valStr := vars[name]
	if strings.TrimSpace(valStr) == "" {
		return "", fmt.Errorf("var '%s' is not in the route", name)
	}

	return valStr, nil
}

type StandardResponse struct {
	StatusCode int         `json:"statusCode"`
	Data       interface{} `json:"data"`
	Error      interface{} `json:"error"`
}

func WriteStandardResponse(w http.ResponseWriter, v interface{}) {
	var resp = StandardResponse{
		StatusCode: http.StatusOK,
		Data:       v,
		Error:      nil,
	}
	writeResponse(w, http.StatusOK, resp)
}

// WriteResponse is a helper function to help write HTTP response
func WriteResponse(w http.ResponseWriter, code int, v interface{}) {
	writeResponse(w, code, v)
}

func writeResponse(w http.ResponseWriter, code int, v interface{}) {
	w.WriteHeader(code)
	log.DebugWithoutCtx("writing response", "kind", reflect.ValueOf(v).Kind(), "content", v)

	if v == nil {
		return
	}

	// Json marshal the resp
	data, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Write the response
	_, err = w.Write(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

// WriteError is a helper function to help write HTTP response
func WriteError(w http.ResponseWriter, code int, err error) {
	writeError(w, code, err)
}

func writeError(w http.ResponseWriter, code int, err error) {

	var errMessage string

	// For Internal errors passed, use a generic message
	if code == http.StatusInternalServerError {
		errMessage = ErrMessageGeneric
	}

	log.ErrorWithoutCtx("Writing error to http response", "error", err)

	// If it a goku error?
	if gErr, ok := errutil.AsGokuError(err); ok {
		errMessage = gErr.GetExternalMsg()
		if code < 1 {
			code = gErr.GetHTTPStatus()
		}
	}

	if errMessage == "" {
		errMessage = err.Error()

	}

	// Still no code? Use InternalServerError
	if code < 1 {
		code = http.StatusInternalServerError
	}

	resp := StandardResponse{
		StatusCode: code,
		Data:       nil,
		Error:      errMessage,
	}

	w.WriteHeader(code)
	data, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("Failed to json.Unmarshal an error for http response: %v", err))
	}
	_, err = w.Write(data)
	if err != nil {
		panic(fmt.Sprintf("Failed to write error to the http response: %v", err))
	}
}

// UnmarshalJSONFromRequest takes in a pointer to an object and populates
// it by reading the content body of the HTTP request, and unmarshaling the
// body into the variable v.
func UnmarshalJSONFromRequest(r *http.Request, v interface{}) error {
	// Read the HTTP request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// api.WriteError(w, http.StatusBadRequest, err, false, nil)
		return err
	}
	defer r.Body.Close()

	if len(body) < 1 {
		// api.WriteError(w, http.StatusBadRequest, api.ErrEmptyBody, false, nil)
		return ErrEmptyBody
	}

	log.DebugWithoutCtx("api: Unmarshaling to JSON", "body", string(body))

	// Unmarshal JSON into Go type
	err = json.Unmarshal(body, &v)
	if err != nil {
		log.ErrorWithoutCtx("api: Unmarshaling to JSON", "error", err)
		return ErrInvalidJSON
	}

	err = validator.Validate(v)
	if err != nil {
		return err
	}

	return nil
}

func HandlerWrapper[ReqT any, RespT any](httpMethod string, fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	switch httpMethod {
	case http.MethodGet:
		return GetGenericGetHandler(fn)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return GetGenericPostPutPatchHandler(fn)
	default:
		// Todo: Implement other method types like DELETE?
	}

	panics.P("HTTP Method type [%s] not implemented by routes.HandlerWrapper().", httpMethod)
	return nil
}

func GetGenericGetHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[HTTP Handler] Handling GET request...")

		// Get the req data from URL
		reqParam, ok := r.URL.Query()["req"]
		if !ok || len(reqParam) < 1 {
			log.Warn(ctx, "An expected URL param is missing", "param", "req")
		}
		if len(reqParam) > 1 {
			WriteError(w, http.StatusBadRequest, fmt.Errorf("multiple URL params with name 'req' found"))
			return
		}

		var req ReqT
		if len(reqParam) == 1 {
			err := json.Unmarshal([]byte(reqParam[0]), &req)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err)
				return
			}
			log.Debug(ctx, "[HTTP Handler] Request unmarshaled from URL", "req", req)
		}

		// Call the method
		resp, err := fn(ctx, req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)
		return
	}
}

func GetGenericPostPutPatchHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[HTTP Handler] Starting...")

		// Get the req from HTTP body
		var req ReqT
		err := httputil.UnmarshalJSONFromRequest(r, &req)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err)
			return
		}

		// Call the method
		resp, err := fn(r.Context(), req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

	}
}

// GetDirectRequestHandler returns a handler which simply forwards the HTTP request to the given method
func GetDirectRequestHandler[RespT any](fn func(context.Context, *http.Request) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[HTTP Handler] Starting...")

		// Call the method
		resp, err := fn(r.Context(), r)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

	}
}
