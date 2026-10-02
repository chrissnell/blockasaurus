// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package querylog

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xERR0R/blocky/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ErrHistoryUnsupportedTarget is returned when the configured query log cannot
// be searched because it is not a local database. Callers turn this into a
// "history is unavailable, and here is why" answer rather than an empty result,
// so the UI can tell the two apart.
var ErrHistoryUnsupportedTarget = errors.New("query log history requires queryLog.type: sqlite")

const (
	// HistoryDefaultLimit is the page size used when the caller does not ask for one.
	HistoryDefaultLimit = 100
	// HistoryMaxLimit caps the page size; a bigger page is clamped, not rejected.
	HistoryMaxLimit = 500

	// historyReadConns bounds the read-only pool. Reads are short and WAL lets
	// them run alongside the writer's flush, so a handful of connections is
	// enough to keep concurrent UI requests from queueing behind each other.
	historyReadConns = 4

	// likeEscape is the ESCAPE character used for user-supplied LIKE patterns.
	likeEscape = `\`

	// BlockedResponseType is the response type of a query blocked by policy. It is
	// the same definition the dashboard's blocked count and the live log's
	// blocked-only toggle use, so the three agree.
	BlockedResponseType = "BLOCKED"
)

// HistoryQuery is a page request against the persisted query log. Every field is
// optional except the time window, which the caller is expected to bound: the
// request_ts index is what keeps a 30-day table responsive, and a substring
// domain match cannot use an index of its own.
type HistoryQuery struct {
	// Client matches the client IP or the resolved client name, as a substring.
	Client string
	// Domain matches the queried name, as a substring.
	Domain string
	// From and To bound request_ts, inclusive of From and exclusive of To. A zero
	// value leaves that end unbounded.
	From time.Time
	To   time.Time
	// BlockedOnly keeps only entries blocked by policy. Same definition the
	// dashboard's blocked count and the live log's toggle use: response type
	// BLOCKED.
	BlockedOnly bool
	// Limit is the page size, clamped to HistoryMaxLimit.
	Limit int
	// Cursor continues a previous page. Empty starts at the newest match.
	Cursor string
}

// HistoryEntry is one row of the persisted query log.
type HistoryEntry struct {
	RequestTS    time.Time
	ClientIP     string
	ClientName   string
	ClientGroup  string
	DurationMs   int64
	Reason       string
	ResponseType string
	QuestionType string
	QuestionName string
	ResponseCode string
	Answer       string
}

// HistoryPage is one page of results, newest first. NextCursor is empty when the
// page is the last one.
type HistoryPage struct {
	Entries    []HistoryEntry
	NextCursor string
}

// HistoryReader answers history queries from the query-log database. It owns its
// own read-only handle rather than borrowing the writer's: the writer serializes
// everything through a single connection so its flush and retention cleanup
// cannot deadlock each other, and sharing that connection would put UI queries
// behind the flush.
type HistoryReader struct {
	db *gorm.DB
}

// NewHistoryReader opens a read-only handle on the configured query-log database.
// It returns ErrHistoryUnsupportedTarget for every query log type that is not a
// local SQLite file, so the caller can report "not configured for history"
// distinctly from "failed to open".
func NewHistoryReader(cfg config.QueryLog) (*HistoryReader, error) {
	if cfg.Type != config.QueryLogTypeSqlite {
		return nil, fmt.Errorf("%w (configured type: %s)", ErrHistoryUnsupportedTarget, cfg.Type)
	}

	dialector, err := newSQLiteReadOnlyDialector(cfg.Target.Reveal())
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("can't open query log database for reading: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("can't access query log read pool: %w", err)
	}

	sqlDB.SetMaxOpenConns(historyReadConns)

	// Belt and suspenders on top of the DSN's mode=ro, matching configstore.Open:
	// enforced at the statement layer, so a regression in URI parsing cannot turn
	// this handle into a second writer.
	if err := db.Exec("PRAGMA query_only=1").Error; err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("can't set query log read handle to query_only: %w", err)
	}

	return &HistoryReader{db: db}, nil
}

// Close releases the read-only handle.
func (r *HistoryReader) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// historyRow is the scan target. rowid is SQLite's implicit primary key: the
// sqlite query log table deliberately has no id column (see databaseMigration),
// so the keyset cursor is built on rowid instead.
//
// request_ts is selected twice. Once as the column, which the driver parses into
// a time.Time off the "datetime" declared type, and once through a CAST that
// strips that declared type so the raw stored text comes back verbatim. The
// cursor carries that text, which makes continuation exact: the driver writes
// timestamps as text in the writer's own location, so rebuilding the comparison
// value from a parsed time would have to reproduce both that location and the
// trimmed fractional seconds to compare equal.
type historyRow struct {
	RowID        int64  `gorm:"column:rowid"`
	RequestTSRaw string `gorm:"column:request_ts_raw"`
	RequestTS    time.Time
	ClientIP     string
	ClientName   string
	ClientGroup  string
	DurationMs   int64
	Reason       string
	ResponseType string
	QuestionType string
	QuestionName string
	ResponseCode string
	Answer       string
}

const historySelect = `rowid AS rowid, CAST(request_ts AS TEXT) AS request_ts_raw, request_ts, ` +
	`client_ip, client_name, client_group, duration_ms, reason, response_type, ` +
	`question_type, question_name, response_code, answer`

// Query returns one page of matching entries, newest first.
func (r *HistoryReader) Query(ctx context.Context, q HistoryQuery) (HistoryPage, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = HistoryDefaultLimit
	}

	limit = min(limit, HistoryMaxLimit)

	tx := r.db.WithContext(ctx).Table("log_entries").Select(historySelect)

	tx, err := applyHistoryFilters(tx, q)
	if err != nil {
		return HistoryPage{}, err
	}

	// Reverse traversal of the request_ts index yields (request_ts, rowid)
	// descending, which is the order the keyset cursor assumes.
	tx = tx.Order("request_ts DESC").Order("rowid DESC").Limit(limit + 1)

	var rows []historyRow
	if err := tx.Find(&rows).Error; err != nil {
		return HistoryPage{}, fmt.Errorf("query log history lookup failed: %w", err)
	}

	// One row past the page size is fetched purely to learn whether another page
	// exists, so a last page never hands back a cursor that returns nothing.
	var nextCursor string

	if len(rows) > limit {
		last := rows[limit-1]
		nextCursor = encodeHistoryCursor(last.RequestTSRaw, last.RowID)
		rows = rows[:limit]
	}

	entries := make([]HistoryEntry, len(rows))
	for i, row := range rows {
		entries[i] = HistoryEntry{
			RequestTS:    row.RequestTS,
			ClientIP:     row.ClientIP,
			ClientName:   row.ClientName,
			ClientGroup:  row.ClientGroup,
			DurationMs:   row.DurationMs,
			Reason:       row.Reason,
			ResponseType: row.ResponseType,
			QuestionType: row.QuestionType,
			QuestionName: row.QuestionName,
			ResponseCode: row.ResponseCode,
			Answer:       row.Answer,
		}
	}

	return HistoryPage{Entries: entries, NextCursor: nextCursor}, nil
}

func applyHistoryFilters(tx *gorm.DB, q HistoryQuery) (*gorm.DB, error) {
	// Timestamps are compared as the text the driver wrote, so the bound value has
	// to be formatted the same way: as a time.Time in the writer's location, which
	// is the local one (entry start times come from time.Now()). The retention
	// cleanup compares the same way.
	if !q.From.IsZero() {
		tx = tx.Where("request_ts >= ?", q.From.Local())
	}

	if !q.To.IsZero() {
		tx = tx.Where("request_ts < ?", q.To.Local())
	}

	if q.BlockedOnly {
		tx = tx.Where("response_type = ?", BlockedResponseType)
	}

	if pattern := likePattern(q.Client); pattern != "" {
		tx = tx.Where(
			"client_ip LIKE ? ESCAPE ? OR client_name LIKE ? ESCAPE ?",
			pattern, likeEscape, pattern, likeEscape,
		)
	}

	if pattern := likePattern(q.Domain); pattern != "" {
		tx = tx.Where("question_name LIKE ? ESCAPE ?", pattern, likeEscape)
	}

	if q.Cursor != "" {
		ts, rowID, err := decodeHistoryCursor(q.Cursor)
		if err != nil {
			return nil, err
		}

		// Compared against the raw stored text, not a re-formatted time, so the
		// predicate stays on the request_ts index.
		tx = tx.Where("request_ts < ? OR (request_ts = ? AND rowid < ?)", ts, ts, rowID)
	}

	return tx, nil
}

// likePattern turns user input into a substring LIKE pattern, escaping the
// wildcards so a domain containing "_" is not a single-character wildcard.
// Returns "" for input that is only whitespace, i.e. no filter.
func likePattern(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	escaped := strings.NewReplacer(
		likeEscape, likeEscape+likeEscape,
		"%", likeEscape+"%",
		"_", likeEscape+"_",
	).Replace(trimmed)

	return "%" + escaped + "%"
}

// ErrInvalidHistoryCursor is returned for a cursor that did not come from a
// previous page.
var ErrInvalidHistoryCursor = errors.New("invalid query log history cursor")

// encodeHistoryCursor packs the last row's position into an opaque token. The
// encoding is deliberately not part of the API contract: callers round-trip it
// unchanged.
func encodeHistoryCursor(requestTSRaw string, rowID int64) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(requestTSRaw + "\x00" + strconv.FormatInt(rowID, 10)),
	)
}

func decodeHistoryCursor(cursor string) (requestTSRaw string, rowID int64, err error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", 0, fmt.Errorf("%w: not a valid token", ErrInvalidHistoryCursor)
	}

	ts, id, found := strings.Cut(string(raw), "\x00")
	if !found || ts == "" {
		return "", 0, fmt.Errorf("%w: malformed token", ErrInvalidHistoryCursor)
	}

	rowID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("%w: malformed row reference", ErrInvalidHistoryCursor)
	}

	return ts, rowID, nil
}
