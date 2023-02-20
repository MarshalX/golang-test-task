package main

// Copied from https://github.com/moul/chizap/blob/main/chizap.go due to issues with suffixes of go-chi's major version
//  ref: https://github.com/moul/chizap/issues/71

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// loggerOpts contains the middleware configuration.
type loggerOpts struct {
	// WithReferer enables logging the "Referer" HTTP header value.
	WithReferer bool

	// WithUserAgent enables logging the "User-Agent" HTTP header value.
	WithUserAgent bool
}

// newLoggerMiddleware returns a logger middleware for chi, that implements the http.Handler interface.
func newLoggerMiddleware(logger *zap.Logger, opts *loggerOpts) func(next http.Handler) http.Handler {
	if logger == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	if opts == nil {
		opts = &loggerOpts{}
	}
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()
			defer func() {
				reqLogger := logger.With(
					zap.String("proto", r.Proto),
					zap.String("path", r.URL.Path),
					zap.String("reqId", middleware.GetReqID(r.Context())),
					zap.Duration("lat", time.Since(t1)),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
				)
				if opts.WithReferer {
					ref := ww.Header().Get("Referer")
					if ref == "" {
						ref = r.Header.Get("Referer")
					}
					if ref != "" {
						reqLogger = reqLogger.With(zap.String("ref", ref))
					}
				}
				if opts.WithUserAgent {
					ua := ww.Header().Get("User-Agent")
					if ua == "" {
						ua = r.Header.Get("User-Agent")
					}
					if ua != "" {
						reqLogger = reqLogger.With(zap.String("ua", ua))
					}
				}
				reqLogger.Info("Served")
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
