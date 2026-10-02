// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package configapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"time"

	"github.com/0xERR0R/blocky/api/configapi"
	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/querylog"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeHistory records the query it was asked for and replays a canned answer,
// so the handler's parameter mapping is asserted without a database.
type fakeHistory struct {
	got  querylog.HistoryQuery
	page querylog.HistoryPage
	err  error
}

func (f *fakeHistory) Query(_ context.Context, q querylog.HistoryQuery) (querylog.HistoryPage, error) {
	f.got = q

	return f.page, f.err
}

var _ = Describe("ConfigAPI query log history", func() {
	var (
		ctx     context.Context
		history *fakeHistory
		h       *configapi.ConfigHandler
	)

	BeforeEach(func() {
		ctx = context.Background()
		history = &fakeHistory{}
		h = configapi.NewConfigHandler(nil, nil, configapi.DoH3Runtime{},
			configapi.QueryLogHistorySource{Reader: history})
	})

	get := func(params configapi.GetQueryLogHistoryParams) configapi.GetQueryLogHistoryResponseObject {
		resp, err := h.GetQueryLogHistory(ctx, configapi.GetQueryLogHistoryRequestObject{Params: params})
		Expect(err).Should(Succeed())

		return resp
	}

	When("no history backend is configured", func() {
		It("answers 503 with the configured reason", func() {
			h = configapi.NewConfigHandler(nil, nil, configapi.DoH3Runtime{},
				configapi.QueryLogHistorySource{UnavailableReason: "queryLog.type is console"})

			resp := get(configapi.GetQueryLogHistoryParams{})
			Expect(resp).Should(BeAssignableToTypeOf(configapi.GetQueryLogHistory503JSONResponse{}))
			Expect(resp.(configapi.GetQueryLogHistory503JSONResponse).Message).
				Should(Equal("queryLog.type is console"))
		})

		It("falls back to a generic reason when none was given", func() {
			h = configapi.NewConfigHandler(nil, nil, configapi.DoH3Runtime{}, configapi.QueryLogHistorySource{})

			resp := get(configapi.GetQueryLogHistoryParams{})
			Expect(resp.(configapi.GetQueryLogHistory503JSONResponse).Message).
				Should(ContainSubstring("sqlite"))
		})
	})

	When("parameters are given", func() {
		It("passes every filter through", func() {
			from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			to := from.Add(24 * time.Hour)
			client, domain, cursor := "192.168.1.5", "example.com", "token"
			blocked := true
			limit := 25

			get(configapi.GetQueryLogHistoryParams{
				Client: &client, Domain: &domain, From: &from, To: &to,
				BlockedOnly: &blocked, Limit: &limit, Cursor: &cursor,
			})

			Expect(history.got).Should(Equal(querylog.HistoryQuery{
				Client: client, Domain: domain, From: from, To: to,
				BlockedOnly: true, Limit: 25, Cursor: cursor,
			}))
		})

		It("leaves omitted filters at their zero value", func() {
			get(configapi.GetQueryLogHistoryParams{})
			Expect(history.got).Should(Equal(querylog.HistoryQuery{}))
		})

		It("rejects a window that is not forward in time", func() {
			from := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
			to := from.Add(-time.Hour)

			resp := get(configapi.GetQueryLogHistoryParams{From: &from, To: &to})
			Expect(resp).Should(BeAssignableToTypeOf(configapi.GetQueryLogHistory400JSONResponse{}))
		})
	})

	When("the reader fails", func() {
		It("maps a rejected cursor to 400", func() {
			history.err = querylog.ErrInvalidHistoryCursor

			resp := get(configapi.GetQueryLogHistoryParams{})
			Expect(resp).Should(BeAssignableToTypeOf(configapi.GetQueryLogHistory400JSONResponse{}))
		})

		It("surfaces anything else as a server error", func() {
			history.err = errors.New("database is locked")

			_, err := h.GetQueryLogHistory(ctx, configapi.GetQueryLogHistoryRequestObject{})
			Expect(err).Should(MatchError(history.err))
		})
	})

	When("the reader returns a page", func() {
		BeforeEach(func() {
			history.page = querylog.HistoryPage{
				Entries: []querylog.HistoryEntry{
					{
						RequestTS:    time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
						ClientIP:     "192.168.1.5",
						ClientName:   "laptop",
						ClientGroup:  "default",
						DurationMs:   12,
						Reason:       "BLOCKED (adlist)",
						ResponseType: "BLOCKED",
						QuestionType: "A",
						QuestionName: "ads.example.net",
						ResponseCode: "NOERROR",
						Answer:       "A (0.0.0.0)",
					},
					{
						RequestTS:    time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC),
						ResponseType: "CACHED",
						QuestionName: "example.com",
					},
				},
				NextCursor: "next",
			}
		})

		It("maps rows onto the live log's field names", func() {
			page := configapi.QueryLogHistoryPage(get(configapi.GetQueryLogHistoryParams{}).(configapi.GetQueryLogHistory200JSONResponse))

			Expect(page.NextCursor).ShouldNot(BeNil())
			Expect(*page.NextCursor).Should(Equal("next"))
			Expect(page.Entries).Should(HaveLen(2))

			first := page.Entries[0]
			Expect(first.ClientIp).Should(Equal("192.168.1.5"))
			Expect(first.ClientNames).Should(Equal("laptop"))
			Expect(first.ClientGroup).Should(Equal("default"))
			Expect(first.DurationMs).Should(BeNumerically("==", 12))
			Expect(first.ResponseReason).Should(Equal("BLOCKED (adlist)"))
			Expect(first.ResponseType).Should(Equal("BLOCKED"))
			Expect(first.Blocked).Should(BeTrue())
			Expect(first.Answer).ShouldNot(BeNil())
			Expect(*first.Answer).Should(Equal("A (0.0.0.0)"))
			Expect(first.Timestamp).Should(BeTemporally("==", history.page.Entries[0].RequestTS))

			// Only response type BLOCKED counts as blocked, the same definition the
			// live log and the dashboard use.
			Expect(page.Entries[1].Blocked).Should(BeFalse())
			Expect(page.Entries[1].Answer).Should(BeNil())
		})

		It("omits the cursor when there is no next page", func() {
			history.page.NextCursor = ""

			page := configapi.QueryLogHistoryPage(get(configapi.GetQueryLogHistoryParams{}).(configapi.GetQueryLogHistory200JSONResponse))
			Expect(page.NextCursor).Should(BeNil())
		})

		It("returns an empty list rather than null when nothing matched", func() {
			history.page = querylog.HistoryPage{}

			page := configapi.QueryLogHistoryPage(get(configapi.GetQueryLogHistoryParams{}).(configapi.GetQueryLogHistory200JSONResponse))
			Expect(page.Entries).ShouldNot(BeNil())
			Expect(page.Entries).Should(BeEmpty())
		})
	})
})

// This one goes over real HTTP against a real query-log database, so the query
// parameter binding the generated router does and the SQL the reader builds are
// both exercised -- the fake above only covers the mapping between them.
var _ = Describe("Config API query log history HTTP integration", func() {
	var (
		srv    *httptest.Server
		dbPath string
	)

	BeforeEach(func() {
		dbPath = filepath.Join(GinkgoT().TempDir(), "querylog.db")

		writerCtx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		writer, err := querylog.NewDatabaseWriter(writerCtx, config.QueryLogTypeSqlite, dbPath, 30, time.Millisecond)
		Expect(err).Should(Succeed())

		now := time.Now().Truncate(time.Second)

		writer.Write(&querylog.LogEntry{
			Start: now.Add(-48 * time.Hour), ClientIP: "192.168.1.9", ClientNames: []string{"old-host"},
			QuestionName: "ancient.example.com", ResponseType: "RESOLVED", QuestionType: "A",
		})
		writer.Write(&querylog.LogEntry{
			Start: now.Add(-2 * time.Hour), ClientIP: "192.168.1.10", ClientNames: []string{"laptop"},
			QuestionName: "ads.example.net", ResponseType: "BLOCKED", QuestionType: "A",
		})
		writer.Write(&querylog.LogEntry{
			Start: now.Add(-time.Hour), ClientIP: "192.168.1.11", ClientNames: []string{"phone"},
			QuestionName: "example.com", ResponseType: "RESOLVED", QuestionType: "AAAA",
		})

		reader, err := querylog.NewHistoryReader(config.QueryLog{
			Type:   config.QueryLogTypeSqlite,
			Target: config.Secret(dbPath),
		})
		Expect(err).Should(Succeed())
		DeferCleanup(reader.Close)

		// The writer flushes on its own ticker; wait for the rows to land before
		// the first read rather than letting each case race it.
		Eventually(func() int {
			page, qErr := reader.Query(context.Background(), querylog.HistoryQuery{})
			Expect(qErr).Should(Succeed())

			return len(page.Entries)
		}).Should(Equal(3))

		router := chi.NewRouter()
		configapi.RegisterEndpoints(router, configapi.NewConfigHandler(nil, nil, configapi.DoH3Runtime{},
			configapi.QueryLogHistorySource{Reader: reader}))
		srv = httptest.NewServer(router)
		DeferCleanup(srv.Close)
	})

	search := func(query string) configapi.QueryLogHistoryPage {
		resp := httpDo(http.MethodGet, srv.URL+"/api/config/query-log/history?"+query, "")
		Expect(resp.StatusCode).Should(Equal(http.StatusOK))

		var page configapi.QueryLogHistoryPage
		decodeBody(resp, &page)

		return page
	}

	names := func(page configapi.QueryLogHistoryPage) []string {
		out := make([]string, len(page.Entries))
		for i, e := range page.Entries {
			out[i] = e.QuestionName
		}

		return out
	}

	It("returns entries newest first", func() {
		Expect(names(search(""))).Should(Equal([]string{
			"example.com", "ads.example.net", "ancient.example.com",
		}))
	})

	It("filters by domain", func() {
		Expect(names(search("domain=ads"))).Should(Equal([]string{"ads.example.net"}))
	})

	It("filters by client name", func() {
		Expect(names(search("client=phone"))).Should(Equal([]string{"example.com"}))
	})

	It("filters by blocked only", func() {
		Expect(names(search("blocked_only=true"))).Should(Equal([]string{"ads.example.net"}))
	})

	It("honours the time window", func() {
		from := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
		Expect(names(search("from=" + url.QueryEscape(from)))).Should(Equal([]string{
			"example.com", "ads.example.net",
		}))
	})

	It("pages with the cursor it issued", func() {
		first := search("limit=2")
		Expect(names(first)).Should(Equal([]string{"example.com", "ads.example.net"}))
		Expect(first.NextCursor).ShouldNot(BeNil())

		second := search("limit=2&cursor=" + url.QueryEscape(*first.NextCursor))
		Expect(names(second)).Should(Equal([]string{"ancient.example.com"}))
		Expect(second.NextCursor).Should(BeNil())
	})

	It("rejects a malformed timestamp", func() {
		resp := httpDo(http.MethodGet, srv.URL+"/api/config/query-log/history?from=yesterday", "")
		Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
		Expect(resp.Body.Close()).Should(Succeed())
	})

	It("rejects a cursor it did not issue", func() {
		resp := httpDo(http.MethodGet, srv.URL+"/api/config/query-log/history?cursor=bogus%21", "")
		Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
		Expect(resp.Body.Close()).Should(Succeed())
	})
})
