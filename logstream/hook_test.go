// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package logstream_test

import (
	"context"

	"github.com/0xERR0R/blocky/logstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

var _ = Describe("Hook", func() {
	var (
		b      *logstream.Broadcaster
		stream <-chan logstream.LogEntry
		logger *logrus.Logger
	)

	BeforeEach(func() {
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		b = logstream.NewBroadcaster(ctx, 100)

		var unsubscribe func()

		stream, unsubscribe = b.Subscribe()
		DeferCleanup(unsubscribe)

		logger, _ = test.NewNullLogger()
		logger.AddHook(logstream.NewHook(b))
	})

	It("broadcasts a log entry with its fields", func() {
		logger.WithField("k", "v").Warn("hello")

		var entry logstream.LogEntry
		Eventually(stream).Should(Receive(&entry))

		Expect(entry.Message).Should(Equal("hello"))
		Expect(entry.Level).Should(Equal("warning"))
		Expect(entry.Fields).Should(HaveKeyWithValue("k", "v"))
	})

	It("broadcasts an entry that carries an unrelated context", func() {
		// The marker rides on Entry.Context, and almost every entry has one: the
		// resolver chain threads its logger through the request context. Only the
		// marker value may suppress a broadcast, never the presence of a context.
		logger.WithContext(context.Background()).Info("not marked")

		Eventually(stream).Should(Receive())
	})

	It("skips entries marked with SkipHook", func() {
		marked := logstream.SkipHook(logger.WithField("prefix", "producer"))

		marked.Info("published by its producer")
		marked.WithField("k", "v").Info("derived entries inherit the marker")

		Consistently(stream).ShouldNot(Receive())
	})

	It("leaves the entry it was given unmarked", func() {
		base := logger.WithField("prefix", "producer")

		logstream.SkipHook(base).Info("skipped")
		base.Info("still broadcast")

		var entry logstream.LogEntry
		Eventually(stream).Should(Receive(&entry))

		Expect(entry.Message).Should(Equal("still broadcast"))
		Consistently(stream).ShouldNot(Receive())
	})
})
