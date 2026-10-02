// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package logstream

import (
	"context"
	"maps"
	"time"

	"github.com/sirupsen/logrus"
)

// skipHookKey marks a logrus entry whose content its producer already published
// to the broadcaster itself. It rides on the entry's context rather than in its
// fields so the marker never reaches the log output.
type skipHookKey struct{}

// skipHookCtx is a constant marker value, built once: it carries no deadline and
// nothing ever cancels it.
//
//nolint:gochecknoglobals
var skipHookCtx = context.WithValue(context.Background(), skipHookKey{}, struct{}{})

// SkipHook returns a copy of logger whose lines Hook.Fire ignores. Entries derived
// from it (WithField, WithFields) inherit the marker; any context the logger
// already carried is replaced.
//
// Use it for log lines whose content the producer publishes to the broadcaster
// itself, which would otherwise be broadcast twice. The query log does: it
// publishes from QueryLoggingResolver so that every queryLog.type reaches the UI,
// not just the console writer.
func SkipHook(logger *logrus.Entry) *logrus.Entry {
	// Set Context on a copy rather than via Entry.WithContext: WithContext clones
	// Data on every call, and this is only ever a marker, never a cancellation
	// scope worth threading a real context for.
	marked := *logger
	marked.Context = skipHookCtx

	return &marked
}

func hookSkipped(entry *logrus.Entry) bool {
	return entry.Context != nil && entry.Context.Value(skipHookKey{}) != nil
}

type Hook struct {
	broadcaster *Broadcaster
}

func NewHook(broadcaster *Broadcaster) *Hook {
	return &Hook{broadcaster: broadcaster}
}

func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *Hook) Fire(entry *logrus.Entry) error {
	if hookSkipped(entry) {
		return nil
	}

	fields := make(map[string]any, len(entry.Data))
	maps.Copy(fields, entry.Data)

	h.broadcaster.Publish(LogEntry{
		Timestamp: entry.Time.UTC().Truncate(time.Millisecond),
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Fields:    fields,
	})

	return nil
}
