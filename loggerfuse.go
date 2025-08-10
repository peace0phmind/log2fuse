// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/peace0phmind/log2fuse/langfuse"
)

// LangfuseLogger a langfuse logger implementation.
type LangfuseLogger struct {
	clock         LoggerClock
	uuidGenerator UUIDGenerator
	logger        *log.Logger
	client        *langfuse.Client
}

// Print logs the record to langfuse and local logger
func (jhl *LangfuseLogger) Print(record *LogRecord) {
	jhl.print(record)
}

func (jhl *LangfuseLogger) print(record *LogRecord) {
	requestBodyText, _ := record.RequestBodyDecoder.decode(record.RequestBody)
	responseBodyText, _ := record.ResponseBodyDecoder.decode(record.ResponseBody)

	// 生成 trace ID 和 span ID
	sessionID := jhl.uuidGenerator.Generate()
	traceID := jhl.uuidGenerator.Generate()
	spanID := jhl.uuidGenerator.Generate()
	startTimestamp := record.StartTime.UTC().Format("2006-01-02T15:04:05.999Z07:00")
	endTimestamp := record.EndTime.UTC().Format("2006-01-02T15:04:05.999Z07:00")

	// 创建 trace 事件
	traceBody := &langfuse.TraceBody{
		ID:        traceID,
		Timestamp: startTimestamp,
		Name:      fmt.Sprintf("%s: %s %s", record.System, record.Method, record.URL),
		Input: map[string]interface{}{
			"url":  record.URL,
			"body": requestBodyText,
		},
		Output: map[string]interface{}{
			"statusCode":   record.StatusCode,
			"responseBody": responseBodyText,
		},
		SessionID: sessionID,
		Tags: []string{
			"http",
			record.System,
			record.Method,
			fmt.Sprintf("status_%d", record.StatusCode),
		},
	}

	// 创建 span 事件（observation）
	spanBody := &langfuse.ObservationBody{
		ID:        spanID,
		TraceID:   traceID,
		Type:      langfuse.ObservationTypeSpan,
		Name:      fmt.Sprintf("%s: %s %s", record.System, record.Method, record.URL),
		StartTime: startTimestamp,
		EndTime:   endTimestamp,
		Input: map[string]interface{}{
			"method":     record.Method,
			"url":        record.URL,
			"proto":      record.Proto,
			"remoteAddr": record.RemoteAddr,
			"headers":    record.RequestHeaders,
			"body":       requestBodyText,
		},
		Output: map[string]interface{}{
			"statusCode":            record.StatusCode,
			"statusText":            http.StatusText(record.StatusCode),
			"responseHeaders":       record.ResponseHeaders,
			"responseBody":          responseBodyText,
			"responseContentLength": record.ResponseContentLength,
			"durationMs":            record.DurationMs,
		},
		Level: langfuse.ObservationLevelDefault,
	}

	// 创建 ingestion 事件
	traceEvent := langfuse.CreateTraceEvent(traceID, startTimestamp, traceBody)
	spanEvent := langfuse.CreateSpanEvent(spanID, startTimestamp, spanBody)

	// 批量发送到 langfuse
	ingestionReq := &langfuse.IngestionRequest{
		Batch: []langfuse.IngestionEvent{*traceEvent, *spanEvent},
		Metadata: map[string]interface{}{
			"source": "log2fuse",
			"system": record.System,
		},
	}

	// 发送到 langfuse
	ctx := context.Background()
	resp, err := jhl.client.Ingest(ctx, ingestionReq)
	if err != nil {
		jhl.logger.Printf("Failed to send to langfuse: %v", err)
		return
	}

	// 记录发送结果
	if len(resp.Successes) > 0 {
		jhl.logger.Printf("Successfully sent %d events to langfuse", len(resp.Successes))
	}
	if len(resp.Errors) > 0 {
		jhl.logger.Printf("Failed to send %d events to langfuse: %+v", len(resp.Errors), resp.Errors)
	}
}
