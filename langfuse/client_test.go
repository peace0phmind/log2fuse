package langfuse

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLangfuseClient(t *testing.T) {
	// 创建客户端
	client := NewClient(
		"http://langfuse.m.com",                      // 或者你的自托管Langfuse实例URL
		"pk-lf-be43e5f5-e011-46bd-bc8f-086135795a0b", // Langfuse Public Key
		"sk-lf-c47b31ea-4045-4e1f-8c11-8343e1cb1bf1", // Langfuse Secret Key
	)

	ctx := context.Background()

	// 1. 检查健康状态
	t.Run("健康检查", func(t *testing.T) {
		fmt.Println("=== 健康检查 ===")
		health, err := client.Health(ctx)
		if err != nil {
			t.Fatalf("健康检查失败: %v", err)
		}
		fmt.Printf("Langfuse版本: %s, 状态: %s\n", health.Version, health.Status)
	})

	// 2. 创建跟踪事件
	t.Run("创建跟踪事件", func(t *testing.T) {
		fmt.Println("\n=== 创建跟踪事件 ===")

		// 创建跟踪体
		traceBody := &TraceBody{
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
		traceEvent := CreateTraceEvent(
			"event-001",
			time.Now().UTC().Format(time.RFC3339),
			traceBody,
		)

		// 验证跟踪事件创建
		if traceEvent == nil {
			t.Fatal("跟踪事件创建失败")
		}
		fmt.Printf("跟踪事件创建成功: %s\n", traceEvent.ID)
	})

	// 3. 创建观察事件
	t.Run("创建观察事件", func(t *testing.T) {
		fmt.Println("=== 创建观察事件 ===")

		// 创建观察体
		observationBody := &ObservationBody{
			ID:        "obs-001",
			TraceID:   "trace-123",
			Type:      ObservationTypeSpan,
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
			Level:       ObservationLevelDefault,
			Environment: "production",
		}

		// 创建观察事件
		observationEvent := CreateObservationEvent(
			"event-002",
			time.Now().UTC().Format(time.RFC3339),
			observationBody,
		)

		// 验证观察事件创建
		if observationEvent == nil {
			t.Fatal("观察事件创建失败")
		}
		fmt.Printf("观察事件创建成功: %s\n", observationEvent.ID)
	})

	// 4. 创建SDK日志事件
	t.Run("创建SDK日志事件", func(t *testing.T) {
		fmt.Println("=== 创建SDK日志事件 ===")

		sdkLogBody := &SDKLogBody{
			Log: map[string]interface{}{
				"level":     "info",
				"message":   "这是一条SDK日志",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}

		sdkLogEvent := CreateSDKLogEvent(
			"event-003",
			time.Now().UTC().Format(time.RFC3339),
			sdkLogBody,
		)

		// 验证SDK日志事件创建
		if sdkLogEvent == nil {
			t.Fatal("SDK日志事件创建失败")
		}
		fmt.Printf("SDK日志事件创建成功: %s\n", sdkLogEvent.ID)
	})

	// 5. 批量发送事件
	t.Run("批量发送事件", func(t *testing.T) {
		fmt.Println("=== 批量发送事件 ===")

		// 重新创建事件用于发送
		traceBody := &TraceBody{
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

		observationBody := &ObservationBody{
			ID:        "obs-001",
			TraceID:   "trace-123",
			Type:      ObservationTypeSpan,
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
			Level:       ObservationLevelDefault,
			Environment: "production",
		}

		sdkLogBody := &SDKLogBody{
			Log: map[string]interface{}{
				"level":     "info",
				"message":   "这是一条SDK日志",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}

		traceEvent := CreateTraceEvent(
			"event-001",
			time.Now().UTC().Format(time.RFC3339),
			traceBody,
		)

		observationEvent := CreateObservationEvent(
			"event-002",
			time.Now().UTC().Format(time.RFC3339),
			observationBody,
		)

		sdkLogEvent := CreateSDKLogEvent(
			"event-003",
			time.Now().UTC().Format(time.RFC3339),
			sdkLogBody,
		)

		ingestionRequest := &IngestionRequest{
			Batch: []IngestionEvent{
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
			t.Fatalf("发送事件失败: %v", err)
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

		// 验证响应
		if len(response.Errors) > 0 {
			t.Logf("有 %d 个事件发送失败", len(response.Errors))
		}

		fmt.Println("\n=== 测试完成 ===")
	})
}

// 测试客户端创建
func TestNewClient(t *testing.T) {
	client := NewClient(
		"http://langfuse.m.com",
		"pk-lf-be43e5f5-e011-46bd-bc8f-086135795a0b",
		"sk-lf-c47b31ea-4045-4e1f-8c11-8343e1cb1bf1",
	)

	if client == nil {
		t.Fatal("客户端创建失败")
	}

	// 注意：由于字段是私有的，我们无法直接访问它们进行测试
	// 这里我们只测试客户端是否成功创建
	t.Log("客户端创建成功")
}

// 测试事件创建函数
func TestCreateTraceEvent(t *testing.T) {
	traceBody := &TraceBody{
		ID:   "test-trace",
		Name: "测试跟踪",
	}

	event := CreateTraceEvent("test-event", "2023-01-01T00:00:00Z", traceBody)

	if event == nil {
		t.Fatal("跟踪事件创建失败")
	}

	if event.ID != "test-event" {
		t.Errorf("期望事件ID为 test-event, 实际为 %s", event.ID)
	}

	if event.Timestamp != "2023-01-01T00:00:00Z" {
		t.Errorf("时间戳不匹配")
	}

	if event.Type != "trace-create" {
		t.Errorf("期望事件类型为 trace-create, 实际为 %s", event.Type)
	}
}

func TestCreateObservationEvent(t *testing.T) {
	observationBody := &ObservationBody{
		ID:      "test-obs",
		TraceID: "test-trace",
		Name:    "测试观察",
	}

	event := CreateObservationEvent("test-event", "2023-01-01T00:00:00Z", observationBody)

	if event == nil {
		t.Fatal("观察事件创建失败")
	}

	if event.ID != "test-event" {
		t.Errorf("期望事件ID为 test-event, 实际为 %s", event.ID)
	}

	if event.Type != "observation-create" {
		t.Errorf("期望事件类型为 observation-create, 实际为 %s", event.Type)
	}
}

func TestCreateSDKLogEvent(t *testing.T) {
	sdkLogBody := &SDKLogBody{
		Log: map[string]interface{}{
			"message": "测试日志",
		},
	}

	event := CreateSDKLogEvent("test-event", "2023-01-01T00:00:00Z", sdkLogBody)

	if event == nil {
		t.Fatal("SDK日志事件创建失败")
	}

	if event.ID != "test-event" {
		t.Errorf("期望事件ID为 test-event, 实际为 %s", event.ID)
	}

	if event.Type != "sdk-log" {
		t.Errorf("期望事件类型为 sdk-log, 实际为 %s", event.Type)
	}
}
