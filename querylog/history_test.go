// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

//go:build !mips && !mipsle && !mips64 && !mips64le && !loong64 && !(netbsd && !amd64) && !(openbsd && !amd64 && !arm64) && !solaris

package querylog

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xERR0R/blocky/config"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HistoryReader", func() {
	var (
		ctx    context.Context
		dbPath string
		reader *HistoryReader
		base   time.Time
	)

	// writeEntries creates a sqlite query log at dbPath, writes the given entries
	// through the real writer (so the schema and the stored timestamp format are
	// the production ones) and waits for the flush.
	writeEntries := func(entries []*LogEntry) {
		writerCtx, cancel := context.WithCancel(ctx)
		DeferCleanup(cancel)

		writer, err := NewDatabaseWriter(writerCtx, config.QueryLogTypeSqlite, dbPath, 30, time.Millisecond)
		Expect(err).Should(Succeed())

		for _, e := range entries {
			writer.Write(e)
		}

		Eventually(func() (int64, error) {
			return countLogEntries(writer.db)
		}).Should(BeNumerically("==", len(entries)))
	}

	entry := func(offset time.Duration, clientIP, clientName, domain, responseType string) *LogEntry {
		return &LogEntry{
			Start:        base.Add(offset),
			ClientIP:     clientIP,
			ClientNames:  []string{clientName},
			ClientGroup:  "default",
			DurationMs:   20,
			ResponseType: responseType,
			ResponseCode: "NOERROR",
			QuestionType: "A",
			QuestionName: domain,
			Answer:       "A (192.0.2.1)",
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		dbPath = filepath.Join(GinkgoT().TempDir(), "querylog.db")
		// Truncated to seconds so the stored text has no fractional part, which is
		// the shape most likely to expose a cursor comparison bug.
		base = time.Now().Truncate(time.Second).Add(-time.Hour)
	})

	openReader := func() {
		var err error

		reader, err = NewHistoryReader(config.QueryLog{
			Type:   config.QueryLogTypeSqlite,
			Target: config.Secret(dbPath),
		})
		Expect(err).Should(Succeed())
		DeferCleanup(reader.Close)
	}

	Describe("opening", func() {
		When("the query log is not a sqlite target", func() {
			It("reports the target as unsupported", func() {
				_, err := NewHistoryReader(config.QueryLog{Type: config.QueryLogTypeConsole})
				Expect(err).Should(MatchError(ErrHistoryUnsupportedTarget))
				Expect(err.Error()).Should(ContainSubstring("console"))
			})
		})

		When("the database file does not exist yet", func() {
			It("fails without creating it", func() {
				_, err := NewHistoryReader(config.QueryLog{
					Type:   config.QueryLogTypeSqlite,
					Target: config.Secret(dbPath),
				})
				Expect(err).Should(HaveOccurred())
				Expect(dbPath).ShouldNot(BeAnExistingFile())
			})
		})

		When("the handle is open", func() {
			It("refuses writes", func() {
				writeEntries([]*LogEntry{entry(0, "192.168.1.10", "laptop", "example.com", "RESOLVED")})
				openReader()

				err := reader.db.Exec("DELETE FROM log_entries").Error
				Expect(err).Should(HaveOccurred())
			})
		})
	})

	Describe("indexes", func() {
		It("indexes the searched columns", func() {
			writeEntries([]*LogEntry{entry(0, "192.168.1.10", "laptop", "example.com", "RESOLVED")})
			openReader()

			var indexed []string
			Expect(reader.db.Raw(
				`SELECT ii.name FROM pragma_index_list('log_entries') il,
				 pragma_index_info(il.name) ii`,
			).Scan(&indexed).Error).Should(Succeed())

			Expect(indexed).Should(ContainElements("request_ts", "client_ip", "client_name", "question_name"))
		})

		// Every one of these is a cost assertion, not a style one. SQLite silently
		// degrades a page to a scan of the whole time window when a predicate stops
		// being an index term, and a temp B-tree for ORDER BY means each page
		// re-sorts the full match set -- which defeats keyset paging entirely. Both
		// are invisible in the results and only show up in the plan.
		DescribeTable("plans every page off the request_ts index, with no sort",
			func(q HistoryQuery) {
				writeEntries([]*LogEntry{entry(0, "192.168.1.10", "laptop", "example.com", "RESOLVED")})
				openReader()

				plan := explainHistoryQuery(reader, q)

				Expect(plan).Should(ContainSubstring("idx_log_entries_request_ts"))
				Expect(plan).ShouldNot(ContainSubstring("TEMP B-TREE"))
			},
			Entry("time-bounded", HistoryQuery{
				From: time.Now().Add(-25 * time.Hour),
				To:   time.Now(),
			}),
			// The keyset predicate has to stay an index bound. Written
			// disjunctively, gorm's separate bind parameters stop SQLite deriving an
			// upper bound and every page scans the window instead.
			Entry("continuing from a cursor", HistoryQuery{
				From:   time.Now().Add(-25 * time.Hour),
				Cursor: encodeHistoryCursor("2026-10-02 12:00:00+00:00", 42),
			}),
			// response_type has its own index and looks cheap to the planner, so
			// without the unary + it wins and drags in a sort.
			Entry("blocked only", HistoryQuery{
				From:        time.Now().Add(-25 * time.Hour),
				BlockedOnly: true,
			}),
		)
	})

	Describe("filtering", func() {
		BeforeEach(func() {
			writeEntries([]*LogEntry{
				entry(0, "192.168.1.10", "laptop", "example.com", "RESOLVED"),
				entry(time.Second, "192.168.1.11", "phone", "ads.example.net", "BLOCKED"),
				entry(2*time.Second, "10.0.0.5", "nas", "bucket_name.example.org", "CACHED"),
			})
			openReader()
		})

		It("returns everything newest first by default", func() {
			page, err := reader.Query(ctx, HistoryQuery{})
			Expect(err).Should(Succeed())
			Expect(page.NextCursor).Should(BeEmpty())
			Expect(domainsOf(page)).Should(Equal([]string{
				"bucket_name.example.org", "ads.example.net", "example.com",
			}))
		})

		It("matches a client by IP substring", func() {
			page, err := reader.Query(ctx, HistoryQuery{Client: "192.168.1.11"})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"ads.example.net"}))
		})

		It("matches a client by resolved name", func() {
			page, err := reader.Query(ctx, HistoryQuery{Client: "nas"})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"bucket_name.example.org"}))
		})

		It("matches a domain substring case-insensitively", func() {
			page, err := reader.Query(ctx, HistoryQuery{Domain: "EXAMPLE.NET"})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"ads.example.net"}))
		})

		It("treats LIKE wildcards in the search text literally", func() {
			page, err := reader.Query(ctx, HistoryQuery{Domain: "bucket_name"})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"bucket_name.example.org"}))

			// "_" is a single-character wildcard in LIKE; escaped, it cannot match
			// the "t." in "bucket.".
			page, err = reader.Query(ctx, HistoryQuery{Domain: "bucket_"})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"bucket_name.example.org"}))

			page, err = reader.Query(ctx, HistoryQuery{Domain: "%"})
			Expect(err).Should(Succeed())
			Expect(page.Entries).Should(BeEmpty())
		})

		It("keeps only blocked entries when asked", func() {
			page, err := reader.Query(ctx, HistoryQuery{BlockedOnly: true})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"ads.example.net"}))
		})

		It("bounds the window with From inclusive and To exclusive", func() {
			page, err := reader.Query(ctx, HistoryQuery{
				From: base.Add(time.Second),
				To:   base.Add(2 * time.Second),
			})
			Expect(err).Should(Succeed())
			Expect(domainsOf(page)).Should(Equal([]string{"ads.example.net"}))
		})

		It("returns the full row", func() {
			page, err := reader.Query(ctx, HistoryQuery{Domain: "ads.example.net"})
			Expect(err).Should(Succeed())
			Expect(page.Entries).Should(HaveLen(1))

			got := page.Entries[0]
			Expect(got.ClientIP).Should(Equal("192.168.1.11"))
			Expect(got.ClientName).Should(Equal("phone"))
			Expect(got.ClientGroup).Should(Equal("default"))
			Expect(got.DurationMs).Should(BeNumerically("==", 20))
			Expect(got.ResponseType).Should(Equal("BLOCKED"))
			Expect(got.ResponseCode).Should(Equal("NOERROR"))
			Expect(got.QuestionType).Should(Equal("A"))
			Expect(got.Answer).Should(Equal("A (192.0.2.1)"))
			Expect(got.RequestTS.UTC()).Should(BeTemporally("==", base.Add(time.Second).UTC()))
		})
	})

	Describe("pagination", func() {
		const total = 25

		BeforeEach(func() {
			entries := make([]*LogEntry, 0, total)
			for i := range total {
				// Every entry shares the same second, so the cursor has to fall back
				// to rowid to make progress at all.
				entries = append(entries, entry(0, "192.168.1.10", "laptop", "example.com", "RESOLVED"))
				entries[i].Answer = "A (192.0.2." + strconv.Itoa(i) + ")"
			}

			writeEntries(entries)
			openReader()
		})

		It("walks every row exactly once across pages", func() {
			seen := make([]string, 0, total)
			cursor := ""

			for pages := 0; ; pages++ {
				Expect(pages).Should(BeNumerically("<", 10), "pagination did not terminate")

				page, err := reader.Query(ctx, HistoryQuery{Limit: 7, Cursor: cursor})
				Expect(err).Should(Succeed())

				for _, e := range page.Entries {
					seen = append(seen, e.Answer)
				}

				cursor = page.NextCursor
				if cursor == "" {
					break
				}
			}

			Expect(seen).Should(HaveLen(total))
			Expect(seen).Should(HaveEach(Not(BeEmpty())))
			Expect(uniq(seen)).Should(HaveLen(total))
		})

		It("omits the cursor on the last page", func() {
			page, err := reader.Query(ctx, HistoryQuery{Limit: total})
			Expect(err).Should(Succeed())
			Expect(page.Entries).Should(HaveLen(total))
			Expect(page.NextCursor).Should(BeEmpty())
		})

		It("clamps the page size", func() {
			page, err := reader.Query(ctx, HistoryQuery{Limit: HistoryMaxLimit + 1000})
			Expect(err).Should(Succeed())
			Expect(page.Entries).Should(HaveLen(total))
		})

		It("rejects a cursor it did not issue", func() {
			_, err := reader.Query(ctx, HistoryQuery{Cursor: "not-a-cursor!!"})
			Expect(errors.Is(err, ErrInvalidHistoryCursor)).Should(BeTrue())

			_, err = reader.Query(ctx, HistoryQuery{Cursor: "YWJj"}) // "abc", no separator
			Expect(errors.Is(err, ErrInvalidHistoryCursor)).Should(BeTrue())
		})
	})
})

// explainHistoryQuery runs EXPLAIN QUERY PLAN over the statement the query path
// actually builds for q.
//
// The placeholders are left as placeholders and the values bound, which is the
// whole point: gorm's Explain() inlines them as literals, and SQLite plans the
// literal form differently -- it can equate two literals it cannot equate two
// bind parameters. Explaining the inlined SQL reports the plan of a statement
// production never executes, and reports the fast one.
func explainHistoryQuery(r *HistoryReader, q HistoryQuery) string {
	stmt := r.db.Session(&gorm.Session{DryRun: true}).Table("log_entries").Select(historySelect)

	stmt, err := applyHistoryFilters(stmt, q)
	Expect(err).Should(Succeed())

	stmt = stmt.Order("request_ts DESC").Order("rowid DESC").Limit(HistoryDefaultLimit).Find(&[]historyRow{})

	// EXPLAIN QUERY PLAN returns four columns and cannot be wrapped in a
	// sub-select, so the rows are read directly rather than scanned into a slice.
	rows, err := r.db.Raw("EXPLAIN QUERY PLAN "+stmt.Statement.SQL.String(), stmt.Statement.Vars...).Rows()
	Expect(err).Should(Succeed())

	defer rows.Close()

	var plan []string

	for rows.Next() {
		var (
			id, parent, notUsed int
			detail              string
		)

		Expect(rows.Scan(&id, &parent, &notUsed, &detail)).Should(Succeed())
		plan = append(plan, detail)
	}

	Expect(rows.Err()).Should(Succeed())

	return strings.Join(plan, "\n")
}

func domainsOf(page HistoryPage) []string {
	out := make([]string, len(page.Entries))
	for i, e := range page.Entries {
		out[i] = e.QuestionName
	}

	return out
}

func uniq(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}
