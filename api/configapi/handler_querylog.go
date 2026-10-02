// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package configapi

import (
	"context"
	"errors"

	"github.com/0xERR0R/blocky/querylog"
)

// QueryLogHistory reads persisted query log entries. An interface so the handler
// can be constructed without a database; *querylog.HistoryReader satisfies it.
type QueryLogHistory interface {
	Query(ctx context.Context, q querylog.HistoryQuery) (querylog.HistoryPage, error)
}

// QueryLogHistorySource is the history backend as seen by the API. Reader is nil
// whenever history cannot be served -- queryLog.type is not a local database, or
// the database could not be opened -- and UnavailableReason says which, so the
// UI can distinguish "not configured" from "no matches".
type QueryLogHistorySource struct {
	Reader            QueryLogHistory
	UnavailableReason string
}

// GetQueryLogHistory serves one page of persisted query log entries.
func (h *ConfigHandler) GetQueryLogHistory(
	ctx context.Context, req GetQueryLogHistoryRequestObject,
) (GetQueryLogHistoryResponseObject, error) {
	if h.history.Reader == nil {
		reason := h.history.UnavailableReason
		if reason == "" {
			reason = querylog.ErrHistoryUnsupportedTarget.Error()
		}

		return GetQueryLogHistory503JSONResponse{ServiceUnavailableJSONResponse{Message: reason}}, nil
	}

	q := historyQueryFromParams(req.Params)

	if !q.To.IsZero() && !q.From.IsZero() && !q.To.After(q.From) {
		return GetQueryLogHistory400JSONResponse{
			BadRequestJSONResponse{Message: "to must be after from"},
		}, nil
	}

	page, err := h.history.Reader.Query(ctx, q)
	if err != nil {
		// A cursor the client made up is a client error, not a server fault.
		if errors.Is(err, querylog.ErrInvalidHistoryCursor) {
			return GetQueryLogHistory400JSONResponse{
				BadRequestJSONResponse{Message: err.Error()},
			}, nil
		}

		return nil, err
	}

	resp := QueryLogHistoryPage{Entries: historyEntriesToAPI(page.Entries)}
	if page.NextCursor != "" {
		resp.NextCursor = &page.NextCursor
	}

	return GetQueryLogHistory200JSONResponse(resp), nil
}

func historyQueryFromParams(params GetQueryLogHistoryParams) querylog.HistoryQuery {
	q := querylog.HistoryQuery{
		BlockedOnly: params.BlockedOnly != nil && *params.BlockedOnly,
	}

	if params.Client != nil {
		q.Client = *params.Client
	}

	if params.Domain != nil {
		q.Domain = *params.Domain
	}

	if params.From != nil {
		q.From = *params.From
	}

	if params.To != nil {
		q.To = *params.To
	}

	if params.Limit != nil {
		q.Limit = *params.Limit
	}

	if params.Cursor != nil {
		q.Cursor = *params.Cursor
	}

	return q
}

func historyEntriesToAPI(entries []querylog.HistoryEntry) []QueryLogHistoryEntry {
	out := make([]QueryLogHistoryEntry, len(entries))

	for i, e := range entries {
		out[i] = QueryLogHistoryEntry{
			Timestamp:      e.RequestTS.UTC(),
			ClientIp:       e.ClientIP,
			ClientNames:    e.ClientName,
			ClientGroup:    e.ClientGroup,
			DurationMs:     e.DurationMs,
			QuestionType:   e.QuestionType,
			QuestionName:   e.QuestionName,
			ResponseCode:   e.ResponseCode,
			ResponseReason: e.Reason,
			ResponseType:   e.ResponseType,
			Blocked:        e.ResponseType == querylog.BlockedResponseType,
		}

		if e.Answer != "" {
			out[i].Answer = &e.Answer
		}
	}

	return out
}
