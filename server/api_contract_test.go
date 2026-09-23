// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/0xERR0R/blocky/auth"
	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/configstore"
	"github.com/0xERR0R/blocky/logstream"
	"github.com/0xERR0R/blocky/pkg/statscollector"

	"github.com/go-chi/chi/v5"
)

// updateAPIContract regenerates the golden file instead of asserting against
// it. Changing the golden is changing the public API contract, so it is a
// deliberate, reviewable act:
//
//	go test ./server -run TestAPIContract -update-api-contract
var updateAPIContract = flag.Bool("update-api-contract", false,
	"rewrite the API contract golden file")

const apiContractGolden = "testdata/api_contract.golden"

// TestAPIContract locks the HTTP surface of the server.
//
// It builds the real production router — the same createHTTPRouter the server
// uses, not a replica — walks every registered route, and records for each one
// the method, the path pattern, and the middleware chain guarding it.
//
// This exists because Blockasaurus is a fork of blocky that is periodically
// merged with upstream (see docs/UPSTREAM_SYNC.md). A merge can silently drop
// one of our endpoints, strip RequireAuth from a route by resolving a
// conflicted Group block toward upstream's brace layout, or import an upstream
// route that collides with ours. None of those show up as a compile error and
// most do not fail an existing test. They do fail here.
//
// What this test does NOT cover, so nobody trusts it further than it reaches:
//
//   - Request and response body shape. TestAPISpecContract below covers the
//     OpenAPI operation set; the schemas behind those operations are not
//     locked by anything.
//   - Fork edits to upstream files generally. See .fork-additions and
//     docs/UPSTREAM_SYNC.md §7.
//   - Middleware behavior. Only the identity and order of the chain is
//     recorded, not what it does.
//
// A diff here is not automatically a bug — but it is always a change to the
// contract the web UI and any API consumer depend on, and it must be
// acknowledged rather than absorbed.
func TestAPIContract(t *testing.T) {
	router := buildContractRouter(t)

	routes, err := walkRoutes(router)
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	assertGolden(t, apiContractGolden, strings.Join(routes, "\n")+"\n",
		"HTTP API contract changed",
		"go test ./server -run TestAPIContract -update-api-contract")
}

// buildContractRouter constructs the production router with every optional
// subsystem present, so no route is conditionally skipped. A nil config store
// or nil broadcaster would silently drop whole route groups and make the
// golden weaker than the thing it is guarding.
func buildContractRouter(t *testing.T) *chi.Mux {
	t.Helper()

	store, err := configstore.Open(filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatalf("open configstore: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	broadcaster := logstream.NewBroadcaster(ctx, 1)
	t.Cleanup(broadcaster.Shutdown)

	collector := statscollector.New()
	t.Cleanup(collector.Close)

	// Start from real defaults rather than a zero Config so that any route
	// pattern derived from a `default:` tag — notably ports.dohPath — is
	// pinned by the golden too.
	defaulted, err := config.WithDefaults[config.Config]()
	if err != nil {
		t.Fatalf("config defaults: %v", err)
	}

	cfg := &defaulted
	cfg.Prometheus.Enable = true

	// metrics.Start mutates the package-global prometheus registry, which
	// server_endpoints_test.go deliberately avoids. We accept that here
	// because the metrics route cannot be captured without calling it, and
	// because Start ignores duplicate-registration errors.
	router := createHTTPRouter(cfg, nil, store, nil, broadcaster, collector, auth.NewWSRevoker())

	// DoH lands on this same mux whenever the admin UI is not on separate
	// listeners (see NewServer). Registration reads only cfg, so a zero
	// Server is sufficient to record the route shape.
	(&Server{}).registerDoHEndpoints(router, cfg)

	return router
}

// allMethods is chi's full method set. A route registered with Handle() (as
// opposed to Get/Post/...) is expanded by chi into one entry per method; we
// collapse those back into a single ANY line so the golden stays readable and
// a real change is not buried under nine identical rows.
var allMethods = []string{
	"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE",
}

type routeKey struct {
	pattern string
	method  string
}

// walkRoutes returns "METHOD /pattern [middleware...]" for every route in the
// tree, sorted by pattern so the golden is stable across registration-order
// changes.
//
// The middleware chain is the reason this is not just a list of paths: moving
// a route out of the authenticated group strips RequireAuth without changing
// its method or path, and that is both the likeliest and the most damaging
// regression an upstream merge can introduce here.
func walkRoutes(router *chi.Mux) ([]string, error) {
	guards := map[routeKey]string{}
	methodsByPattern := map[string]map[string]bool{}

	walk := func(method, route string, _ http.Handler,
		middlewares ...func(http.Handler) http.Handler,
	) error {
		if methodsByPattern[route] == nil {
			methodsByPattern[route] = map[string]bool{}
		}

		methodsByPattern[route][method] = true
		guards[routeKey{route, method}] = middlewareChain(middlewares)

		return nil
	}

	if err := chi.Walk(router, walk); err != nil {
		return nil, err
	}

	patterns := make([]string, 0, len(methodsByPattern))
	for pattern := range methodsByPattern {
		patterns = append(patterns, pattern)
	}

	sort.Strings(patterns)

	var routes []string

	for _, pattern := range patterns {
		methods := byPatternMethods(methodsByPattern[pattern])

		if isAllMethods(methodsByPattern[pattern]) {
			routes = append(routes, formatRoute("ANY", pattern, guards[routeKey{pattern, methods[0]}]))

			continue
		}

		for _, m := range methods {
			routes = append(routes, formatRoute(m, pattern, guards[routeKey{pattern, m}]))
		}
	}

	return routes, nil
}

func formatRoute(method, pattern, guard string) string {
	return fmt.Sprintf("%-7s %-45s %s", method, pattern, guard)
}

// middlewareChain renders the guards on a route by function name, in order.
// Closures returned by middleware constructors resolve to stable names like
// "auth.RequireAuth.func1", which is enough to tell "authenticated" from
// "public" and to notice a guard being dropped or reordered.
func middlewareChain(middlewares []func(http.Handler) http.Handler) string {
	if len(middlewares) == 0 {
		return "[public]"
	}

	names := make([]string, 0, len(middlewares))

	for _, mw := range middlewares {
		full := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
		names = append(names, middlewareName(full))
	}

	return "[" + strings.Join(names, " ") + "]"
}

// middlewareName reduces a runtime symbol like
// "github.com/0xERR0R/blocky/server.registerUIRoutes.func1.RequireAuth.2" to
// the middleware it actually is: "RequireAuth". The trailing segments are
// closure indices the compiler assigns, and the leading ones are whichever
// function happened to install the middleware — neither is part of the
// contract, and both churn on unrelated edits.
func middlewareName(symbol string) string {
	segments := strings.Split(symbol[strings.LastIndex(symbol, "/")+1:], ".")

	for i := len(segments) - 1; i >= 0; i-- {
		if seg := segments[i]; !closureSegment(seg) {
			return seg
		}
	}

	return symbol
}

func closureSegment(seg string) bool {
	if strings.HasPrefix(seg, "func") {
		seg = strings.TrimPrefix(seg, "func")
	}

	if seg == "" {
		return true
	}

	for _, r := range seg {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func byPatternMethods(methods map[string]bool) []string {
	named := make([]string, 0, len(methods))
	for m := range methods {
		named = append(named, m)
	}

	sort.Strings(named)

	return named
}

func isAllMethods(methods map[string]bool) bool {
	if len(methods) != len(allMethods) {
		return false
	}

	for _, m := range allMethods {
		if !methods[m] {
			return false
		}
	}

	return true
}

// assertGolden compares got against the golden file, or rewrites it when
// -update-api-contract is set. The failure message names the added and removed
// lines rather than dumping both files, because during an upstream merge the
// question is always "what moved?", not "what does the whole surface look
// like?".
func assertGolden(t *testing.T, path, got, what, regenCmd string) {
	t.Helper()

	if *updateAPIContract {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}

		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Logf("wrote %s", path)

		return
	}

	wantBytes, err := os.ReadFile(path) //nolint:gosec // fixed test fixture path
	if err != nil {
		t.Fatalf("read golden (regenerate with -update-api-contract): %v", err)
	}

	if got == string(wantBytes) {
		return
	}

	added, removed := diffLines(string(wantBytes), got)

	var b strings.Builder

	fmt.Fprintf(&b, "%s.\n\n", what)

	if len(removed) > 0 {
		b.WriteString("GONE (a consumer relying on these breaks):\n")

		for _, r := range removed {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}

	if len(added) > 0 {
		b.WriteString("NEW:\n")

		for _, r := range added {
			fmt.Fprintf(&b, "  + %s\n", r)
		}
	}

	b.WriteString("\nIf this change is intended, regenerate the golden deliberately:\n")
	fmt.Fprintf(&b, "  %s\n", regenCmd)
	b.WriteString("and call out the contract change in the pull request.\n")

	t.Fatal(b.String())
}

func diffLines(want, got string) (added, removed []string) {
	inWant := map[string]bool{}
	for _, l := range strings.Split(strings.TrimSpace(want), "\n") {
		inWant[l] = true
	}

	inGot := map[string]bool{}

	for _, l := range strings.Split(strings.TrimSpace(got), "\n") {
		inGot[l] = true

		if !inWant[l] {
			added = append(added, l)
		}
	}

	for l := range inWant {
		if !inGot[l] {
			removed = append(removed, l)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	return added, removed
}
