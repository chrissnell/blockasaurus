// Modified by Chris Snell, 2026
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type httpServer struct {
	inner http.Server

	name string
}

const (
	serverReadTimeout       = 30 * time.Second
	serverReadHeaderTimeout = 10 * time.Second
	serverWriteTimeout      = 60 * time.Second
)

func newHTTPServer(name string, handler http.Handler) *httpServer {
	return &httpServer{
		inner: http.Server{
			ReadTimeout:       serverReadTimeout,
			ReadHeaderTimeout: serverReadHeaderTimeout,
			WriteTimeout:      serverWriteTimeout,
			Handler:           withCommonMiddleware(handler),
		},

		name: name,
	}
}

func (s *httpServer) String() string {
	return s.name
}

func (s *httpServer) Serve(ctx context.Context, l net.Listener) error {
	go func() {
		<-ctx.Done()

		s.inner.Close()
	}()

	if err := s.inner.Serve(l); err != nil {
		return fmt.Errorf("HTTP server '%s' failed to serve: %w", s.name, err)
	}

	return nil
}

func withCommonMiddleware(inner http.Handler) *chi.Mux {
	// Middleware must be defined before routes, so
	// create a new router and mount the inner handler
	mux := chi.NewMux()

	mux.Use(
		secureHeadersMiddleware,
		newCORSMiddleware(),
	)

	mux.Mount("/", inner)

	return mux
}

type httpMiddleware = func(http.Handler) http.Handler

func secureHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("x-xss-protection", "1; mode=block")
		}

		next.ServeHTTP(w, r)
	})
}

func newCORSMiddleware() httpMiddleware {
	const corsMaxAge = 5 * time.Minute

	options := cors.Options{
		// Same-origin only: mirror the Origin header iff its host matches
		// the request Host. The SPA is served from the same origin as the
		// API, so no cross-origin credentialed fetches should ever be
		// legitimate. AllowedOrigins: ["*"] + AllowCredentials: true is
		// spec-invalid (browsers reject it), and a permissive
		// AllowOriginFunc defeats the CSRF defense provided by
		// SameSite=Lax + the X-Requested-With header check.
		AllowOriginFunc:  sameOriginFunc,
		AllowCredentials: true,
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		ExposedHeaders:   []string{"Link"},
		MaxAge:           int(corsMaxAge.Seconds()),
	}

	return cors.New(options).Handler
}

// sameOriginFunc returns true iff the Origin header's host matches the
// request Host. Returns false for empty/unparseable Origin values.
func sameOriginFunc(r *http.Request, origin string) bool {
	if origin == "" {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	return strings.EqualFold(u.Host, r.Host)
}
