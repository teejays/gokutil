package gopi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/teejays/gokutil/errutil"
	"github.com/teejays/gokutil/log"
)

var llog = log.GetLogger().WithHeading("gopi")

// Route represents a standard route object
type Route struct {
	Method       string
	Version      int
	Path         string
	HandlerFunc  http.HandlerFunc
	Authenticate bool
}

type MiddlewareFunc func(next http.Handler) http.Handler

type MiddlewareFuncs struct {
	PreMiddlewares  []mux.MiddlewareFunc
	AuthMiddleware  mux.MiddlewareFunc
	PostMiddlewares []mux.MiddlewareFunc
}

type Server struct {
	rootHandler http.Handler
}

func NewServer(ctx context.Context, routes []Route, middlewares MiddlewareFuncs, pathPrefix string) (Server, error) {
	m, err := GetHandler(ctx, routes, middlewares, pathPrefix)
	if err != nil {
		return Server{}, fmt.Errorf("could not setup the http handler: %w", err)
	}

	return Server{rootHandler: m}, nil

}

// StartServer initializes and runs the HTTP server. This is a blocking function.
func (s *Server) StartServer(ctx context.Context, addr string, port int) error {

	http.Handle("/", s.rootHandler)

	// Start the server
	llog.Info(ctx, "[HTTP server listening", "address", addr, "port", port)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", addr, port), nil)
	if err != nil {
		return fmt.Errorf("HTTP Server failed to start or continue running: %w", err)
	}

	return nil
}

// GetHandler constructs a HTTP handler with all the routes and middleware funcs configured
func GetHandler(ctx context.Context, routes []Route, middlewares MiddlewareFuncs, pathPrefix string) (http.Handler, error) {

	// Initiate a router
	m := mux.NewRouter()
	if pathPrefix != "" {
		// Remove any leading or trailing slashes and then add a leading slash
		pathPrefix = "/" + strings.Trim(pathPrefix, "/")
		m = m.PathPrefix(pathPrefix).Subrouter()
	}

	// Standard/Common routes

	// NotFound handler:
	// Make sure non-matching requests are also logged, and a specific response is returned
	m.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llog.Debug(ctx, "Request received for non-existent route", "method", r.Method, "path", r.URL.String())

		err := errutil.NewGerror("Route not configured: [%s]", r.URL.String()).
			SetHTTPStatus(http.StatusNotFound).
			SetExternalMsg("You have reached an API server which is developed and maintained by Ongoku but the route accessed [%s] is non-existent.", r.URL.String())

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		writeError(w, http.StatusNotFound, err)
	})

	// Ping route:
	m.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		WriteStandardResponse(w, "pong")
	}).Methods(http.MethodGet)

	// Enable CORS
	// TODO: Have tighter control over CORS policy, but okay for
	// as long as we're just developing. This shouldn't really go on prod.
	originsOk := handlers.AllowedOrigins([]string{"*"})
	credsOk := handlers.AllowCredentials()
	headersOk := handlers.AllowedHeaders([]string{"Content-Type", "authorization"})
	methodsOk := handlers.AllowedMethods([]string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch})
	corsEnabler := handlers.CORS(originsOk, credsOk, headersOk, methodsOk)

	// Register routes to the handler
	// Set up pre handler middlewares
	m.Use(PaniRecovercMiddleware) // Use it first so that it can recover from panics that happen in other middlewares
	for _, mw := range middlewares.PreMiddlewares {
		m.Use(mux.MiddlewareFunc(mw))
	}

	// Create an authenticated subrouter
	var a *mux.Router
	if middlewares.AuthMiddleware != nil {
		a = m.PathPrefix("").Subrouter()
		a.Use(mux.MiddlewareFunc(middlewares.AuthMiddleware))
	}

	// Validate Routes
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes provided")
	}
	var routesLookup = map[string]bool{}
	for _, r := range routes {
		key := r.Method + " " + GetRoutePattern(r)
		if routesLookup[key] {
			return nil, fmt.Errorf("multiple routes provided for [%s]", key)
		}
		routesLookup[key] = true
	}
	// Range over routes and register them
	for _, route := range routes {
		// If the route is supposed to be authenticated, use auth mux
		r := m.NewRoute().Subrouter()

		if route.Authenticate {
			if a == nil {
				// We marked a route as requiring authentication but provided no auth middleware func :(
				return nil, fmt.Errorf("route for %s has authentication flag set but no authentication middleware has been provided", route.Path)
			}
			r = a
		}
		// Register the route
		llog.None(ctx, "Registering endpoint", "method", route.Method, "path", GetRoutePattern(route))

		if route.Method == "" {
			return nil, fmt.Errorf("route [%s] has no http method", route.Path)
		}
		if route.Path == "" {
			return nil, fmt.Errorf("route [%s] has no path", route.Path)
		}
		if route.HandlerFunc == nil {
			return nil, fmt.Errorf("route [%s] has no HandlerFunc", route.Path)
		}

		mRoute := r.HandleFunc(GetRoutePattern(route), route.HandlerFunc).Methods(route.Method)

		fullPath, err := mRoute.GetPathTemplate()
		if err != nil {
			return nil, err
		}
		llog.Debug(ctx, "Registered Endpoint", "method", route.Method, "path", fullPath)
	}

	// Set up pre handler middlewares
	for _, mw := range middlewares.PostMiddlewares {
		m.Use(mux.MiddlewareFunc(mw))
	}

	mc := corsEnabler(m)

	return mc, nil
}
func PaniRecovercMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func() {
			if r := recover(); r != nil {
				llog.Error(ctx, "Panic while handling the request", "panic", r)
				err := errutil.New("Panic while handling the request: %s", r)
				writeError(w, http.StatusInternalServerError, err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggerMiddleware is a http.Handler middleware function that logs any request received
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Log the request
		llog.Debug(ctx, "START: HTTP request", "method", r.Method, "path", r.URL.String())
		// Call the next handler
		next.ServeHTTP(w, r)
		llog.Debug(ctx, "END: HTTP request", "method", r.Method, "path", r.URL.String())
	})
}

// SetJSONHeaderMiddleware sets the header for the response
func SetJSONHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		next.ServeHTTP(w, r)

		// Set the header after the underlying handler has run (but only do it if it's not already set)
		if w.Header().Get("Content-Type") == "" {
			log.Info(ctx, "Setting JSON header", "key", "Content-Type", "value", "application/json; charset=UTF-8")
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		}
	})
}

// returns the url match pattern for the route
func GetRoutePattern(r Route) string {
	// remove any leading or trailing slashes
	r.Path = strings.Trim(r.Path, "/")
	return fmt.Sprintf("/v%d/%s", r.Version, r.Path)
}
