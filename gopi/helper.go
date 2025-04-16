package gopi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/httputil"
	"github.com/teejays/gokutil/log"
	"github.com/teejays/gokutil/panics"
	"github.com/teejays/gokutil/validate"

	"github.com/teejays/gokutil/gopi/json"
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

type StandardResponseGeneric[T any] struct {
	StatusCode int    `json:"statusCode"`
	Data       T      `json:"data"`
	Error      string `json:"error"`
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
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	log.InfoWithoutCtx("[writeResponse] Content type set", "key", "Content-Type", "value", "application/json; charset=UTF-8")

	w.WriteHeader(code) // Calling write header can usually mean we cannot set the headers now, so all headers must be set before this

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

func HandlerWrapper[ReqT any, RespT any](httpMethod string, fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	switch httpMethod {
	case http.MethodGet:
		return GetGenericGetHandler(fn)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return GetGenericPostPutPatchHandler(fn)
	case http.MethodDelete:
		return GetGenericGetHandler(fn) // because DELETE requests don't have a body but rely on URL params like GET
	default:
		// Todo: Implement other method types?
	}

	panics.P("HTTP Method type [%s] not implemented by routes.HandlerWrapper().", httpMethod)
	return nil
}

func GetGenericGetHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[START] HTTP Handle GetDelete")

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
			// Strict Unmarshal so we don't mess up things.
			err := json.UnmarshalStrict([]byte(reqParam[0]), &req)
			if err != nil {
				WriteError(w, http.StatusBadRequest, err)
				return
			}
			log.Debug(ctx, "[HTTP Handler] Request unmarshaled from URL", "req", json.MustPrettyPrint(req))
			// Validate the request
			err = validate.Struct(req)
			if err != nil {
				err = errutil.Wrap(err, "Validating the request param")
				WriteError(w, http.StatusBadRequest, err)
				return
			}
		}

		// Call the method
		resp, err := fn(ctx, req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

		log.Debug(ctx, "[END] HTTP Handle GetDelete")

		return
	}
}

func GetGenericPostPutPatchHandler[ReqT, RespT any](fn func(context.Context, ReqT) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[START] HTTP Handle PostPutPatch")

		// Get the req from HTTP body
		var req ReqT
		err := httputil.UnmarshalJSONFromRequest(r, &req)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err)
			return
		}

		log.Debug(ctx, "[HTTP Handler] Request unmarshaled from body", "req", json.MustPrettyPrint(req))

		// Call the method
		resp, err := fn(r.Context(), req)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

		log.Debug(ctx, "[END] HTTP Handle PostPutPatch")

	}
}

// GetDirectRequestHandler returns a handler which simply forwards the HTTP request to the given method
func GetDirectRequestHandler[RespT any](fn func(context.Context, *http.Request) (RespT, error)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug(ctx, "[START] HTTP Handle DirectRequest")

		// Call the method
		resp, err := fn(r.Context(), r)
		if err != nil {
			WriteError(w, 0, err)
			return
		}

		WriteStandardResponse(w, resp)

		log.Debug(ctx, "[END] HTTP Handle DirectRequest")

	}
}

type AuthorizationHeaderInfo struct {
	Scheme string
	Token  string
}

var ErrNoAuthorizationHeader = errors.New("no authorization header found")
var ErrInvalidAuthorizationHeader = errors.New("authorization header has an unexpected format: it's not 'Authorization: <SCHEME> <TOKEN>'")

// ExtractAuthorizationHeaderFromRequest extracts the authorization header from the HTTP request alongside the scheme and token.
// The Authorization header value should be like: <scheme e.g. Bearer, ApiKey> <token>.
// A non-error guarantees that the scheme and token are not empty.
func ExtractAuthorizationHeaderFromRequest(r *http.Request) (AuthorizationHeaderInfo, error) {

	// Get the authentication header
	val := r.Header.Get("Authorization")
	log.Debug(r.Context(), "Extracting token from HTTP request", "Authorization", val)
	val = strings.TrimSpace(val)

	// No auth header
	if val == "" {
		return AuthorizationHeaderInfo{}, ErrNoAuthorizationHeader
	}

	// - split by the space
	valParts := strings.Split(val, " ")
	if len(valParts) != 2 {
		return AuthorizationHeaderInfo{}, errutil.Wrap(ErrInvalidAuthorizationHeader, "invalid number of parts found in authorization header: expected %d, got %d", 2, len(valParts))
	}

	if valParts[0] == "" || valParts[1] == "" {
		return AuthorizationHeaderInfo{}, errutil.Wrap(ErrInvalidAuthorizationHeader, "empty scheme or token found in authorization header")
	}

	return AuthorizationHeaderInfo{
		Scheme: valParts[0],
		Token:  valParts[1],
	}, nil
}
