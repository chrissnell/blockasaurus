package querylog

import (
	"reflect"
	"strings"

	"github.com/0xERR0R/blocky/log"
	"github.com/0xERR0R/blocky/logstream"
	"github.com/sirupsen/logrus"
)

const loggerPrefixLoggerWriter = "queryLog"

// ResolvedMessage is the log message for a resolved query. The console writer and
// the UI log stream share it so an entry reads the same on both.
const ResolvedMessage = "query resolved"

type LoggerWriter struct {
	logger *logrus.Entry
}

func NewLoggerWriter() *LoggerWriter {
	// QueryLoggingResolver publishes every entry to the UI log stream itself, for
	// all query log types. Mark these lines so the logstream hook doesn't publish
	// them a second time.
	return &LoggerWriter{logger: logstream.SkipHook(log.PrefixedLog(loggerPrefixLoggerWriter))}
}

func (d *LoggerWriter) Write(entry *LogEntry) {
	d.logger.WithFields(LogEntryFields(entry)).Info(ResolvedMessage)
}

func (d *LoggerWriter) CleanUp() {
	// Nothing to do
}

func LogEntryFields(entry *LogEntry) logrus.Fields {
	return withoutZeroes(logrus.Fields{
		"client_ip":       entry.ClientIP,
		"client_names":    strings.Join(entry.ClientNames, "; "),
		"client_group":    entry.ClientGroup,
		"response_reason": entry.ResponseReason,
		"response_type":   entry.ResponseType,
		"response_code":   entry.ResponseCode,
		"question_name":   entry.QuestionName,
		"question_type":   entry.QuestionType,
		"answer":          entry.Answer,
		"duration_ms":     entry.DurationMs,
		"instance":        entry.BlockyInstance,
	})
}

func withoutZeroes(fields logrus.Fields) logrus.Fields {
	for k, v := range fields {
		if reflect.ValueOf(v).IsZero() {
			delete(fields, k)
		}
	}

	return fields
}
