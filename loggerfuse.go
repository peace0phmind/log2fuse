// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/peace0phmind/log2fuse/langfuse"
)

// LangfuseLogger a langfuse logger implementation.
type LangfuseLogger struct {
	clock         LoggerClock
	uuidGenerator UUIDGenerator
	logger        *log.Logger
	client        *langfuse.Client
	chain         chan *LogRecord
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewLangfuseLogger creates a new LangfuseLogger instance
func NewLangfuseLogger(clock LoggerClock, uuidGenerator UUIDGenerator, logger *log.Logger, client *langfuse.Client) *LangfuseLogger {
	ctx, cancel := context.WithCancel(context.Background())

	jhl := &LangfuseLogger{
		clock:         clock,
		uuidGenerator: uuidGenerator,
		logger:        logger,
		client:        client,
		chain:         make(chan *LogRecord, 1000), // 设置chain大小为1000
		ctx:           ctx,
		cancel:        cancel,
	}

	// 启动后台处理goroutine
	jhl.startProcessor()

	return jhl
}

// Print logs the record to langfuse and local logger
func (jhl *LangfuseLogger) Print(record *LogRecord) {
	// 直接将记录加入chain，不进行阻塞
	select {
	case jhl.chain <- record:
		// 成功加入chain
	default:
		// chain已满，移除最早的消息并加入新消息
		select {
		case <-jhl.chain: // 移除最早的消息
			jhl.chain <- record // 加入新消息
		default:
			// 如果还是失败，记录错误
			jhl.logger.Printf("Failed to add record to chain, chain is full")
		}
	}
}

// startProcessor starts the background processor for handling chain records
func (jhl *LangfuseLogger) startProcessor() {
	go func() {
		jhl.processChain()
	}()
}

// processChain processes records from the chain
func (jhl *LangfuseLogger) processChain() {
	for {
		select {
		case record := <-jhl.chain:
			jhl.processRecord(record)
		case <-jhl.ctx.Done():
			return
		}
	}
}

// processRecord processes a single record with retry mechanism
func (jhl *LangfuseLogger) processRecord(record *LogRecord) {
	maxRetries := 3
	retryDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 检查是否已取消
		select {
		case <-jhl.ctx.Done():
			return
		default:
		}

		// 检查langfuse client状态
		if !jhl.isClientHealthy() {
			jhl.logger.Printf("Langfuse client is not healthy, retrying in %v", retryDelay)

			select {
			case <-time.After(retryDelay):
			case <-jhl.ctx.Done():
				return
			}

			retryDelay *= 2 // 指数退避
			continue
		}

		// 尝试发送记录
		if err := jhl.sendRecord(record); err != nil {
			jhl.logger.Printf("Failed to send record (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				select {
				case <-time.After(retryDelay):
				case <-jhl.ctx.Done():
					return
				}
				retryDelay *= 2
				continue
			}
			// 最后一次尝试失败，记录错误
			jhl.logger.Printf("Failed to send record after %d attempts: %v", maxRetries, err)
		}
		return
	}
}

// isClientHealthy checks if the langfuse client is healthy
func (jhl *LangfuseLogger) isClientHealthy() bool {
	ctx, cancel := context.WithTimeout(jhl.ctx, 5*time.Second)
	defer cancel()

	health, err := jhl.client.Health(ctx)
	if err != nil {
		jhl.logger.Printf("check client healthy: %v", err)
		return false
	}
	return health.Status == "OK"
}

// sendRecord sends a single record to langfuse
func (jhl *LangfuseLogger) sendRecord(record *LogRecord) error {
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
	resp, err := jhl.client.Ingest(jhl.ctx, ingestionReq)
	if err != nil {
		return fmt.Errorf("failed to send to langfuse: %w", err)
	}

	// 记录发送结果
	if len(resp.Successes) > 0 {
		jhl.logger.Printf("Successfully sent %d events to langfuse", len(resp.Successes))
	}
	if len(resp.Errors) > 0 {
		jhl.logger.Printf("Failed to send %d events to langfuse: %+v", len(resp.Errors), resp.Errors)
	}

	return nil
}

// Stop stops the background processor
func (jhl *LangfuseLogger) Stop() {
	jhl.cancel()
}

// Close closes the logger and cleans up resources
func (jhl *LangfuseLogger) Close() {
	jhl.Stop()
	close(jhl.chain)
}
