package querylog

import (
	"context"
	"time"

	"github.com/0xERR0R/blocky/logstream"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("LoggerWriter", func() {
	Describe("logger query log", func() {
		When("New log entry was created", func() {
			It("should be logged", func() {
				writer := NewLoggerWriter()
				logger, hook := test.NewNullLogger()
				writer.logger = logger.WithField("k", "v")

				writer.Write(&LogEntry{
					Start:      time.Now(),
					DurationMs: 20,
				})

				Expect(hook.Entries).Should(HaveLen(1))
				Expect(hook.LastEntry().Message).Should(Equal(ResolvedMessage))
			})

			It("should not publish to the log stream itself", func() {
				// QueryLoggingResolver publishes for every query log type, so this
				// writer must leave the stream alone -- otherwise the console type
				// would show every query twice.
				ctx, cancel := context.WithCancel(context.Background())
				DeferCleanup(cancel)

				broadcaster := logstream.NewBroadcaster(ctx, 10)
				stream, unsubscribe := broadcaster.Subscribe()
				DeferCleanup(unsubscribe)

				writer := NewLoggerWriter()

				// Swap only the sink, keeping the marker the constructor attached.
				logger, hook := test.NewNullLogger()
				logger.AddHook(logstream.NewHook(broadcaster))
				writer.logger.Logger = logger

				writer.Write(&LogEntry{
					Start:      time.Now(),
					DurationMs: 20,
				})

				Expect(hook.Entries).Should(HaveLen(1))
				Expect(stream).Should(BeEmpty())

				// The marker must not leak into the log output either.
				Expect(hook.LastEntry().Data).Should(Equal(logrus.Fields{
					"prefix":      loggerPrefixLoggerWriter,
					"duration_ms": int64(20),
				}))
			})
		})
		When("Cleanup is called", func() {
			It("should do nothing", func() {
				writer := NewLoggerWriter()
				writer.CleanUp()
			})
		})
	})

	Describe("LogEntryFields", func() {
		It("should return log fields", func() {
			entry := LogEntry{
				ClientIP:     "ip",
				DurationMs:   100,
				QuestionType: "qtype",
				ResponseCode: "rcode",
			}

			fields := LogEntryFields(&entry)

			Expect(fields).Should(HaveKeyWithValue("client_ip", entry.ClientIP))
			Expect(fields).Should(HaveKeyWithValue("duration_ms", entry.DurationMs))
			Expect(fields).Should(HaveKeyWithValue("question_type", entry.QuestionType))
			Expect(fields).Should(HaveKeyWithValue("response_code", entry.ResponseCode))

			Expect(fields).ShouldNot(HaveKey("client_names"))
			Expect(fields).ShouldNot(HaveKey("question_name"))
		})
	})

	DescribeTable("withoutZeroes",
		func(value any, isZero bool) {
			fields := withoutZeroes(logrus.Fields{"a": value})

			if isZero {
				Expect(fields).Should(BeEmpty())
			} else {
				Expect(fields).ShouldNot(BeEmpty())
			}
		},
		Entry("empty string",
			"",
			true),
		Entry("non-empty string",
			"something",
			false),
		Entry("zero int",
			0,
			true),
		Entry("non-zero int",
			1,
			false),
	)
})
