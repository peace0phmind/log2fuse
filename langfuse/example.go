package langfuse

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ExampleUsage 展示如何使用生成的客户端代码
func ExampleUsage() {
	// 创建客户端实例
	client, err := NewClient("https://cloud.langfuse.com")
	if err != nil {
		log.Fatal(err)
	}

	// 设置基本认证
	client.RequestEditors = append(client.RequestEditors, func(ctx context.Context, req *http.Request) error {
		req.SetBasicAuth("your-public-key", "your-secret-key")
		return nil
	})

	// 创建带响应的客户端
	clientWithResponses, err := NewClientWithResponses("https://cloud.langfuse.com")
	if err != nil {
		log.Fatal(err)
	}

	// 设置基本认证
	clientWithResponses.ClientInterface.(*Client).RequestEditors = append(
		clientWithResponses.ClientInterface.(*Client).RequestEditors,
		func(ctx context.Context, req *http.Request) error {
			req.SetBasicAuth("your-public-key", "your-secret-key")
			return nil
		},
	)

	// 示例：创建评分
	ctx := context.Background()

	// 创建评分请求
	scoreValue := CreateScoreValue{}
	scoreValue.FromCreateScoreValue0(0.95) // 设置数值评分

	scoreId := "score-123"
	comment := "High accuracy score"
	scoreRequest := CreateScoreRequest{
		Id:      &scoreId,
		Name:    "accuracy",
		Value:   scoreValue,
		Comment: &comment,
	}

	// 发送请求
	resp, err := clientWithResponses.ScoreCreateWithResponse(ctx, scoreRequest)
	if err != nil {
		log.Printf("Error creating score: %v", err)
		return
	}

	if resp.JSON200 != nil {
		fmt.Printf("Score created successfully: %+v\n", resp.JSON200)
	} else {
		fmt.Printf("Error response: %d - %s\n", resp.HTTPResponse.StatusCode, resp.Body)
	}

	// 示例：获取评分列表
	limit := 10
	page := 1
	scoreResp, err := clientWithResponses.ScoreV2GetWithResponse(ctx, &ScoreV2GetParams{
		Limit: &limit,
		Page:  &page,
	})
	if err != nil {
		log.Printf("Error getting scores: %v", err)
		return
	}

	if scoreResp.JSON200 != nil {
		fmt.Printf("Scores retrieved: %+v\n", scoreResp.JSON200)
	} else {
		fmt.Printf("Error getting scores: %d - %s\n", scoreResp.HTTPResponse.StatusCode, scoreResp.Body)
	}
}

// ExampleWithCustomTransport 展示如何使用自定义传输层
func ExampleWithCustomTransport() {
	// 创建自定义传输层
	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// 创建带响应的客户端
	clientWithResponses, err := NewClientWithResponses(
		"https://cloud.langfuse.com",
		WithHTTPClient(httpClient),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 设置认证
	clientWithResponses.ClientInterface.(*Client).RequestEditors = append(
		clientWithResponses.ClientInterface.(*Client).RequestEditors,
		func(ctx context.Context, req *http.Request) error {
			req.SetBasicAuth("your-public-key", "your-secret-key")
			return nil
		},
	)

	fmt.Println("Client created with custom transport")
}

// ExampleHealthCheck 展示健康检查
func ExampleHealthCheck() {
	clientWithResponses, err := NewClientWithResponses("https://cloud.langfuse.com")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	resp, err := clientWithResponses.HealthHealthWithResponse(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}

	if resp.JSON200 != nil {
		fmt.Printf("Health status: %+v\n", resp.JSON200)
	} else {
		fmt.Printf("Health check error: %d - %s\n", resp.HTTPResponse.StatusCode, resp.Body)
	}
}

// ExampleGetTraces 展示获取 traces
func ExampleGetTraces() {
	clientWithResponses, err := NewClientWithResponses("https://cloud.langfuse.com")
	if err != nil {
		log.Fatal(err)
	}

	// 设置认证
	clientWithResponses.ClientInterface.(*Client).RequestEditors = append(
		clientWithResponses.ClientInterface.(*Client).RequestEditors,
		func(ctx context.Context, req *http.Request) error {
			req.SetBasicAuth("your-public-key", "your-secret-key")
			return nil
		},
	)

	ctx := context.Background()
	limit := 10
	page := 1
	resp, err := clientWithResponses.TraceListWithResponse(ctx, &TraceListParams{
		Limit: &limit,
		Page:  &page,
	})
	if err != nil {
		log.Printf("Error getting traces: %v", err)
		return
	}

	if resp.JSON200 != nil {
		fmt.Printf("Traces count: %d\n", len(resp.JSON200.Data))
		for i, trace := range resp.JSON200.Data {
			if i >= 3 { // 只显示前3个
				break
			}
			fmt.Printf("  Trace %d: ID=%s, Name=%s\n", i+1, trace.Id, *trace.Name)
		}
	} else {
		fmt.Printf("Error getting traces: %d - %s\n", resp.HTTPResponse.StatusCode, resp.Body)
	}
}
