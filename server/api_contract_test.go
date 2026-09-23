// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/0xERR0R/blocky/auth"
	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/configstore"
	"github.com/0xERR0R/blocky/logstream"
	"github.com/0xERR0R/blocky/pkg/statscollector"

	"net/http"

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
// uses, not a replica — walks every registered route, and compares the full
// method+pattern inventory against a checked-in golden file.
//
// This exists because Blockasaurus is a fork of blocky that is periodically
// merged with upstream (see docs/UPSTREAM_SYNC.md). A merge can silently drop
// one of our endpoints, move a route between the public and authenticated
// groups, or import an upstream route that collides with ours. None of those
// show up as a compile error and most do not fail an existing test. They do
// fail here.
//
// A diff in this test is not automatically a bug — but it is always a change
// to the contract the web UI and any API consumer depend on, and it must be
// acknowledged rather than absorbed.
func TestAPIContract(t *testing.T) {
	router := buildContractRouter(t)

	routes, err := walkRoutes(router)
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	got := strings.Join(routes, "\n") + "\n"

	if *updateAPIContract {
		if err := os.MkdirAll(filepath.Dir(apiContractGolden), 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}

		if err := os.WriteFile(apiContractGolden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Logf("wrote %s (%d routes)", apiContractGolden, len(routes))

		return
	}

	wantBytes, err := os.ReadFile(apiContractGolden)
	if err != nil {
		t.Fatalf("read golden (regenerate with -update-api-contract): %v", err)
	}

	want := string(wantBytes)
	if got == want {
		return
	}

	added, removed := diffLines(want, got)

	var b strings.Builder

	b.WriteString("HTTP API contract changed.\n\n")

	if len(removed) > 0 {
		b.WriteString("Routes REMOVED (a consumer that calls these now gets 404):\n")

		for _, r := range removed {
			fmt.Fprintf(&b, "  - %s\n", r)
		}
	}

	if len(added) > 0 {
		b.WriteString("Routes ADDED:\n")

		for _, r := range added {
			fmt.Fprintf(&b, "  + %s\n", r)
		}
	}

	b.WriteString("\nIf this change is intended, regenerate the golden deliberately:\n")
	b.WriteString("  go test ./server -run TestAPIContract -update-api-contract\n")
	b.WriteString("and call out the contract change in the pull request.\n")

	t.Fatal(b.String())
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

	cfg := &config.Config{}
	cfg.Prometheus.Enable = true
	cfg.Prometheus.Path = "/metrics"

	// openAPIImpl is nil: RegisterOpenAPIEndpoints registers the generated
	// routes from the OpenAPI spec regardless of the implementation behind
	// them, and this test asserts the route inventory, not handler behavior.
	return createHTTPRouter(cfg, nil, store, nil, broadcaster, collector, auth.NewWSRevoker())
}

// allMethods is chi's full method set. A route registered with Handle() (as
// opposed to Get/Post/...) is expanded by chi into one entry per method; we
// collapse those back into a single ANY line so the golden stays readable and
// a real change is not buried under nine identical rows.
var allMethods = []string{
	"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE",
}

// walkRoutes returns "METHOD /pattern" for every route in the tree, sorted by
// pattern so the golden is stable across registration-order changes.
func walkRoutes(router *chi.Mux) ([]string, error) {
	byPattern := map[string]map[string]bool{}

	walk := func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if byPattern[route] == nil {
			byPattern[route] = map[string]bool{}
		}

		byPattern[route][method] = true

		return nil
	}

	if err := chi.Walk(router, walk); err != nil {
		return nil, err
	}

	patterns := make([]string, 0, len(byPattern))
	for pattern := range byPattern {
		patterns = append(patterns, pattern)
	}

	sort.Strings(patterns)

	var routes []string

	for _, pattern := range patterns {
		methods := byPattern[pattern]

		if isAllMethods(methods) {
			routes = append(routes, fmt.Sprintf("%-7s %s", "ANY", pattern))

			continue
		}

		named := make([]string, 0, len(methods))
		for m := range methods {
			named = append(named, m)
		}

		sort.Strings(named)

		for _, m := range named {
			routes = append(routes, fmt.Sprintf("%-7s %s", m, pattern))
		}
	}

	return routes, nil
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
