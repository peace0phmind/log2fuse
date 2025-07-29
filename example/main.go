package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/peace0phmind/log2fuse/langfuse"
)

func main() {
	// 创建客户端
	client := langfuse.NewClient(
		"http://langfuse.m.com",                      // 或者你的自托管Langfuse实例URL
		"pk-lf-be43e5f5-e011-46bd-bc8f-086135795a0b", // Langfuse Public Key
		"sk-lf-c47b31ea-4045-4e1f-8c11-8343e1cb1bf1", // Langfuse Secret Key
	)

	ctx := context.Background()

	// 1. 检查健康状态
	fmt.Println("=== 健康检查 ===")
	health, err := client.Health(ctx)
	if err != nil {
		log.Fatalf("健康检查失败: %v", err)
	}
	fmt.Printf("Langfuse版本: %s, 状态: %s\n", health.Version, health.Status)

	// 2. 创建跟踪事件
	fmt.Println("\n=== 创建跟踪事件 ===")

	// 创建跟踪体
	traceBody := &langfuse.TraceBody{
		ID:          "trace-123",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Name:        "示例跟踪",
		UserID:      "user-456",
		Input:       map[string]interface{}{"prompt": "Hello, world!"},
		Output:      map[string]interface{}{"response": "Hi there!"},
		SessionID:   "session-789",
		Release:     "1.0.0",
		Version:     "1.0.0",
		Metadata:    map[string]interface{}{"source": "example"},
		Tags:        []string{"example", "demo"},
		Environment: "production",
		Public:      true,
	}

	// 创建跟踪事件
	traceEvent := langfuse.CreateTraceEvent(
		"event-001",
		time.Now().UTC().Format(time.RFC3339),
		traceBody,
	)

	// 3. 创建观察事件
	fmt.Println("=== 创建观察事件 ===")

	// 创建观察体
	observationBody := &langfuse.ObservationBody{
		ID:        "obs-001",
		TraceID:   "trace-123",
		Type:      langfuse.ObservationTypeSpan,
		Name:      "LLM调用",
		StartTime: time.Now().UTC().Format(time.RFC3339),
		EndTime:   time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339),
		Model:     "gpt-4",
		ModelParameters: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  1000,
		},
		Input:       map[string]interface{}{"prompt": "Hello, world!"},
		Output:      map[string]interface{}{"response": "Hi there!"},
		Level:       langfuse.ObservationLevelDefault,
		Environment: "production",
	}

	// 创建观察事件
	observationEvent := langfuse.CreateObservationEvent(
		"event-002",
		time.Now().UTC().Format(time.RFC3339),
		observationBody,
	)

	// 4. 创建SDK日志事件
	fmt.Println("=== 创建SDK日志事件 ===")

	sdkLogBody := &langfuse.SDKLogBody{
		Log: map[string]interface{}{
			"level":     "info",
			"message":   "这是一条SDK日志",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	sdkLogEvent := langfuse.CreateSDKLogEvent(
		"event-003",
		time.Now().UTC().Format(time.RFC3339),
		sdkLogBody,
	)

	// 5. 批量发送事件
	fmt.Println("=== 批量发送事件 ===")

	ingestionRequest := &langfuse.IngestionRequest{
		Batch: []langfuse.IngestionEvent{
			*traceEvent,
			*observationEvent,
			*sdkLogEvent,
		},
		Metadata: map[string]interface{}{
			"sdk_version": "1.0.0",
			"source":      "example",
		},
	}

	response, err := client.Ingest(ctx, ingestionRequest)
	if err != nil {
		log.Fatalf("发送事件失败: %v", err)
	}

	// 6. 处理响应
	fmt.Println("=== 响应结果 ===")
	fmt.Printf("成功事件数: %d\n", len(response.Successes))
	fmt.Printf("失败事件数: %d\n", len(response.Errors))

	for _, success := range response.Successes {
		fmt.Printf("成功事件 ID: %s, 状态码: %d\n", success.ID, success.Status)
	}

	for _, error := range response.Errors {
		fmt.Printf("失败事件 ID: %s, 状态码: %d, 消息: %s\n",
			error.ID, error.Status, error.Message)
	}

	fmt.Println("\n=== 示例完成 ===")
}
