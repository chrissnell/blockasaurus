// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"io"

	"github.com/0xERR0R/blocky/api/configapi"
	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/querylog"
)

// newQueryLogHistorySource opens the read-only handle the History tab of the
// Logs page reads from, and returns the closer that owns it (nil when none was
// opened).
//
// A failure here is never fatal: the query log itself keeps writing and the live
// log keeps streaming, so the only consequence is that history cannot be
// searched. The reason travels to the UI so the tab can say why it is empty.
// Only the two "this can never work as configured" reasons are passed through
// verbatim -- neither message contains a path. An open failure is logged in full
// and reported generically, since its message does carry the database path.
func newQueryLogHistorySource(cfg config.QueryLog) (configapi.QueryLogHistorySource, io.Closer) {
	reader, err := querylog.NewHistoryReader(cfg)
	if err == nil {
		return configapi.QueryLogHistorySource{Reader: reader}, reader
	}

	if errors.Is(err, querylog.ErrHistoryUnsupportedTarget) ||
		errors.Is(err, querylog.ErrHistoryUnsupportedPlatform) {
		return configapi.QueryLogHistorySource{UnavailableReason: err.Error()}, nil
	}

	logger().WithError(err).Warn("query log history is unavailable")

	return configapi.QueryLogHistorySource{
		UnavailableReason: "the query log database could not be opened for reading; see the server log",
	}, nil
}
