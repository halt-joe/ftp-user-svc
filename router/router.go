package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/halt-joe/ftp-user-svc/apierror"
	"github.com/halt-joe/ftp-user-svc/handlers"
	log "github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// set default to 200 OK status code.
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// FTPLoginName - name used for login end point route
const FTPLoginName = "FTPLogin"

// logger - creates an HTTP handler that logs incoming requests
func logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// add requestId to request's context
		ctx := r.Context()
		id := uuid.New()
		ctx = context.WithValue(ctx, apierror.ContextKeyRequestID, id.String())

		r = r.WithContext(ctx)

		// get new response writer to capture response status
		lrw := newLoggingResponseWriter(w)

		username := "Unknown"
		if name == FTPLoginName {
			username = handlers.GetUserNameFromLoginRequest(r)
		}

		inner.ServeHTTP(lrw, r)

		reqID := r.Context().Value(apierror.ContextKeyRequestID)

		log.Info(fmt.Sprintf(
			"%s %s %s %s %s %s %d %s",
			reqID,
			r.RemoteAddr,
			r.Method,
			r.RequestURI,
			name,
			username,
			lrw.statusCode,
			time.Since(start),
		))
	})
}

func makeRoute(router *mux.Router, method string, path string, name string, handler http.HandlerFunc) {
	var httpHandler http.Handler

	httpHandler = handler
	router.Methods(method).
		Path(path).
		Name(name).
		Handler(logger(httpHandler, name))
}

// Create - Instantiates a router and loads the routes
func Create(env *handlers.Env) *mux.Router {
	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	router := mux.NewRouter().StrictSlash(true)

	router.Methods("GET").
		Path("/metrics").
		Name("Metrics").
		Handler(logger(sentryHandler.Handle(promhttp.Handler()), "Metrics"))
	makeRoute(router, "DELETE", "/ftpusers/{id}", "FTPUserDelete", sentryHandler.HandleFunc(env.IDDelete))
	makeRoute(router, "POST", "/login", FTPLoginName, sentryHandler.HandleFunc(env.LoginHandler))
	makeRoute(router, "DELETE", "/mappings/{system}/{id}", "MappingsSystemIDDelete", sentryHandler.HandleFunc(env.SystemIDDelete))
	makeRoute(router, "GET", "/mappings/{system}/{id}", "MappingsSystemIDGet", sentryHandler.HandleFunc(env.SystemIDGet))
	makeRoute(router, "POST", "/mappings/{system}", "MappingsSystemPost", sentryHandler.HandleFunc(env.SystemPost))
	makeRoute(router, "GET", "/ftpusers", "FTPUsersGet", sentryHandler.HandleFunc(env.Get))
	makeRoute(router, "GET", "/ftpusers/{id}", "FTPUserGet", sentryHandler.HandleFunc(env.IDGet))
	makeRoute(router, "POST", "/ftpusers", "FTPUsersPost", sentryHandler.HandleFunc(env.Post))
	makeRoute(router, "PUT", "/ftpusers/{id}", "FTPUserPut", sentryHandler.HandleFunc(env.IDPut))
	makeRoute(router, "PATCH", "/ftpusers/{id}", "FTPUserPatch", sentryHandler.HandleFunc(env.IDPatch))
	makeRoute(router, "GET", "/mappings/{system}", "MappingsSystemGet", sentryHandler.HandleFunc(env.SystemGet))

	return router
}
