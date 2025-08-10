// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/peace0phmind/log2fuse/langfuse"
)

type clockContextKey string

// ClockContextKey can be used to fake time.
const ClockContextKey clockContextKey = "clock"

// LoggerClock is the source of current time.
type LoggerClock interface {
	Now() time.Time
}

// SystemLoggerClock uses OS system time.
type SystemLoggerClock struct{}

// Now returns current OS system time.
func (*SystemLoggerClock) Now() time.Time {
	return time.Now()
}

type uuidGeneratorContextKey string

// UUIDGeneratorContextKey can be used to fake UUID generator.
const UUIDGeneratorContextKey uuidGeneratorContextKey = "uuid-generator"

// UUIDGenerator is a UUID generator strategy.
type UUIDGenerator interface {
	Generate() string
}

// RandomUUIDGenerator generates secure random UUID v4.
type RandomUUIDGenerator struct{}

// Generate generates secure random UUID.
func (g *RandomUUIDGenerator) Generate() string {
	return Must(GenerateUUID4()).String()
}

// EmptyUUIDGenerator returns empty string.
type EmptyUUIDGenerator struct{}

// Generate returns empty string.
func (g *EmptyUUIDGenerator) Generate() string {
	return ""
}

type logWriterContextKey string

// LogWriterContextKey can be used to spy log writes.
const LogWriterContextKey logWriterContextKey = "log-writer"

// LogWriter is a write strategy.
type LogWriter interface {
	Write(log string) error
}

// FileLogWriter writes logs to a File (like stdout).
type FileLogWriter struct {
	file *os.File
}

func (w *FileLogWriter) Write(log string) error {
	_, err := w.file.WriteString(log)
	return err
}

func createJSONHTTPLogger(ctx context.Context, config *Config, logger *log.Logger) *JSONHTTPLogger {
	clock := createClock(ctx)
	uuidGenerator := createUUIDGenerator(ctx, config)
	externalLogWriter, hasExternalLogWriter := ctx.Value(LogWriterContextKey).(LogWriter)
	if hasExternalLogWriter {
		return &JSONHTTPLogger{clock: clock, uuidGenerator: uuidGenerator, logger: logger, writer: externalLogWriter}
	}
	return &JSONHTTPLogger{clock: clock, uuidGenerator: uuidGenerator, logger: logger, writer: &FileLogWriter{file: os.Stdout}}
}

func createLangfuseLogger(ctx context.Context, config *Config, logger *log.Logger, client *langfuse.Client) *LangfuseLogger {
	clock := createClock(ctx)
	uuidGenerator := createUUIDGenerator(ctx, config)
	return &LangfuseLogger{clock: clock, uuidGenerator: uuidGenerator, logger: logger, client: client}
}

func createUUIDGenerator(ctx context.Context, config *Config) UUIDGenerator {
	if config.GenerateLogID {
		externalUUIDGenerator, hasExternalUUIDGenerator := ctx.Value(UUIDGeneratorContextKey).(UUIDGenerator)
		if hasExternalUUIDGenerator {
			return externalUUIDGenerator
		}
		return &RandomUUIDGenerator{}
	}
	return &EmptyUUIDGenerator{}
}

func createClock(ctx context.Context) LoggerClock {
	externalClock, hasExternalClock := ctx.Value(ClockContextKey).(LoggerClock)
	if hasExternalClock {
		return externalClock
	}
	return &SystemLoggerClock{}
}
