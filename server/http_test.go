package server

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP middleware", func() {
	var handler http.Handler

	BeforeEach(func() {
		handler = withCommonMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	})

	// Upstream serves the API to arbitrary origins (AllowedOrigins: ["*"],
	// wildcard request headers, Private Network Access preflights). This fork
	// serves a cookie-authenticated admin UI from the same origin as the API,
	// so its policy is same-origin only: "*" plus AllowCredentials is
	// spec-invalid, and a permissive origin check would defeat the
	// SameSite=Lax + X-Requested-With CSRF defense.
	Describe("CORS", func() {
		// httptest.NewRequest sets Host to example.com, so this is the
		// request's own origin.
		const sameOrigin = "https://example.com"

		preflight := func(origin string, headers map[string]string) *http.Response {
			req := httptest.NewRequest(http.MethodOptions, "/api/blocking/disable", nil)
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)

			if origin != "" {
				req.Header.Set("Origin", origin)
			}

			for k, v := range headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			return rec.Result()
		}

		It("answers a same-origin preflight, mirroring the origin", func() {
			res := preflight(sameOrigin, nil)

			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(Equal(sameOrigin))
			Expect(res.Header.Get("Access-Control-Allow-Credentials")).Should(Equal("true"))
			Expect(res.Header.Get("Access-Control-Allow-Methods")).Should(ContainSubstring(http.MethodGet))
		})

		It("allows the methods and headers the UI actually sends", func() {
			res := preflight(sameOrigin, map[string]string{
				"Access-Control-Request-Method":  http.MethodPut,
				"Access-Control-Request-Headers": "content-type,x-csrf-token,x-requested-with",
			})

			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(Equal(sameOrigin))
			// rs/cors echoes the requested header names as the client spelled them
			Expect(res.Header.Get("Access-Control-Allow-Headers")).Should(ContainSubstring("x-csrf-token"))
			Expect(res.Header.Get("Access-Control-Allow-Headers")).Should(ContainSubstring("x-requested-with"))
		})

		It("refuses a cross-origin preflight", func() {
			res := preflight("https://grafana.example.com", nil)

			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(BeEmpty())
			Expect(res.Header.Get("Access-Control-Allow-Credentials")).Should(BeEmpty())
		})

		It("refuses a preflight with no Origin header", func() {
			res := preflight("", nil)

			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("refuses a preflight with an unparseable Origin", func() {
			res := preflight("://nonsense", nil)

			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(BeEmpty())
		})

		It("does not opt in to Private Network Access preflights", func() {
			// Chromium sends this when a public site addresses a private IP.
			// Answering it would let a hosted page reach the admin API on a LAN.
			res := preflight(sameOrigin, map[string]string{
				"Access-Control-Request-Private-Network": "true",
			})

			Expect(res.Header.Get("Access-Control-Allow-Private-Network")).Should(BeEmpty())
		})

		It("mirrors the origin on an actual same-origin request", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
			req.Header.Set("Origin", sameOrigin)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			Expect(res.Header.Get("Access-Control-Allow-Origin")).Should(Equal(sameOrigin))
			Expect(res.Header.Get("Access-Control-Expose-Headers")).Should(ContainSubstring("Link"))
		})
	})
})
