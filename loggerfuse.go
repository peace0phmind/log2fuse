// Package log2fuse a Traefik HTTP logger plugin.
package log2fuse

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
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

	// 健康状态管理
	healthMutex sync.RWMutex
	isHealthy   bool
	lastError   time.Time
	probeMode   bool
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
		isHealthy:     true, // 初始状态假设为健康
	}

	// 启动后台处理goroutine
	jhl.startProcessor()

	// 启动健康探测
	jhl.StartHealthProbe()

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

		// 尝试发送记录
		if err := jhl.sendRecord(record); err != nil {
			jhl.logger.Printf("Failed to send record (attempt %d/%d): %v", attempt+1, maxRetries, err)

			// 标记为不健康状态，进入探测模式
			jhl.markUnhealthy()

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
		} else {
			// 发送成功，标记为健康状态
			jhl.markHealthy()
			return
		}
	}
}

// markUnhealthy marks the client as unhealthy and enters probe mode
func (jhl *LangfuseLogger) markUnhealthy() {
	jhl.healthMutex.Lock()
	defer jhl.healthMutex.Unlock()

	jhl.isHealthy = false
	jhl.lastError = jhl.clock.Now()
	jhl.probeMode = true
}

// markHealthy marks the client as healthy and exits probe mode
func (jhl *LangfuseLogger) markHealthy() {
	jhl.healthMutex.Lock()
	defer jhl.healthMutex.Unlock()

	jhl.isHealthy = true
	jhl.probeMode = false
}

// isClientHealthy checks if the langfuse client is healthy
// This method is now only called when in probe mode or when explicitly needed
func (jhl *LangfuseLogger) isClientHealthy() bool {
	jhl.healthMutex.RLock()
	defer jhl.healthMutex.RUnlock()

	// 如果当前状态是健康的，直接返回true
	if jhl.isHealthy && !jhl.probeMode {
		return true
	}

	// 在探测模式下，进行实际的健康检查
	if jhl.probeMode {
		// 检查距离上次错误是否已经过了足够的时间（避免频繁检查）
		if time.Since(jhl.lastError) < 5*time.Second {
			return false
		}

		// 执行实际的健康检查
		return jhl.performHealthCheck()
	}

	return jhl.isHealthy
}

// performHealthCheck performs the actual health check against langfuse
func (jhl *LangfuseLogger) performHealthCheck() bool {
	ctx, cancel := context.WithTimeout(jhl.ctx, 5*time.Second)
	defer cancel()

	health, err := jhl.client.Health(ctx)
	if err != nil {
		jhl.logger.Printf("Health check failed: %v", err)
		return false
	}

	isHealthy := health.Status == "OK"
	if isHealthy {
		jhl.logger.Printf("Langfuse client recovered, exiting probe mode")
	}

	return isHealthy
}

// sendRecord sends a single record to langfuse
func (jhl *LangfuseLogger) sendRecord(record *LogRecord) error {
	// 在探测模式下，发送前检查健康状态
	if jhl.isInProbeMode() {
		if !jhl.isClientHealthy() {
			return fmt.Errorf("client is unhealthy and in probe mode")
		}
		// 健康检查通过，退出探测模式
		jhl.markHealthy()
	}

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

// isInProbeMode checks if the client is currently in probe mode
func (jhl *LangfuseLogger) isInProbeMode() bool {
	jhl.healthMutex.RLock()
	defer jhl.healthMutex.RUnlock()
	return jhl.probeMode
}

// StartHealthProbe starts a background health probe when in probe mode
func (jhl *LangfuseLogger) StartHealthProbe() {
	go func() {
		ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if jhl.isInProbeMode() {
					if jhl.isClientHealthy() {
						jhl.logger.Printf("Health probe detected recovery, exiting probe mode")
						break
					}
				} else {
					// 不在探测模式下，继续等待
					continue
				}
			case <-jhl.ctx.Done():
				return
			}
		}
	}()
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
