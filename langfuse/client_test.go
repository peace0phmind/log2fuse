package langfuse

import (
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	// 测试创建基本客户端
	client, err := NewClient("https://cloud.langfuse.com")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("Client is nil")
	}
	if client.Server != "https://cloud.langfuse.com/" {
		t.Errorf("Expected server URL to be 'https://cloud.langfuse.com/', got '%s'", client.Server)
	}
}

func TestNewClientWithResponses(t *testing.T) {
	// 测试创建带响应的客户端
	clientWithResponses, err := NewClientWithResponses("https://cloud.langfuse.com")
	if err != nil {
		t.Fatalf("Failed to create client with responses: %v", err)
	}
	if clientWithResponses == nil {
		t.Fatal("ClientWithResponses is nil")
	}
	if clientWithResponses.ClientInterface == nil {
		t.Fatal("ClientInterface is nil")
	}
}

func TestCreateScoreValue(t *testing.T) {
	// 测试创建数值评分
	scoreValue := CreateScoreValue{}
	err := scoreValue.FromCreateScoreValue0(0.95)
	if err != nil {
		t.Fatalf("Failed to set numeric score value: %v", err)
	}

	// 测试获取数值评分
	value, err := scoreValue.AsCreateScoreValue0()
	if err != nil {
		t.Fatalf("Failed to get numeric score value: %v", err)
	}
	if value != 0.95 {
		t.Errorf("Expected score value to be 0.95, got %f", value)
	}

	// 测试创建字符串评分
	scoreValue2 := CreateScoreValue{}
	err = scoreValue2.FromCreateScoreValue1("excellent")
	if err != nil {
		t.Fatalf("Failed to set string score value: %v", err)
	}

	// 测试获取字符串评分
	strValue, err := scoreValue2.AsCreateScoreValue1()
	if err != nil {
		t.Fatalf("Failed to get string score value: %v", err)
	}
	if strValue != "excellent" {
		t.Errorf("Expected score value to be 'excellent', got '%s'", strValue)
	}
}

func TestCreateScoreRequest(t *testing.T) {
	// 测试创建评分请求
	scoreValue := CreateScoreValue{}
	scoreValue.FromCreateScoreValue0(0.85)

	scoreId := "test-score-123"
	comment := "Test score"
	request := CreateScoreRequest{
		Id:      &scoreId,
		Name:    "test_accuracy",
		Value:   scoreValue,
		Comment: &comment,
	}

	if request.Name != "test_accuracy" {
		t.Errorf("Expected name to be 'test_accuracy', got '%s'", request.Name)
	}
	if request.Id == nil || *request.Id != "test-score-123" {
		t.Error("Expected ID to be 'test-score-123'")
	}
	if request.Comment == nil || *request.Comment != "Test score" {
		t.Error("Expected comment to be 'Test score'")
	}
}

func TestClientWithCustomHTTPClient(t *testing.T) {
	// 测试使用自定义 HTTP 客户端
	clientWithResponses, err := NewClientWithResponses(
		"https://cloud.langfuse.com",
		WithHTTPClient(&http.Client{}),
	)
	if err != nil {
		t.Fatalf("Failed to create client with custom HTTP client: %v", err)
	}
	if clientWithResponses == nil {
		t.Fatal("ClientWithResponses is nil")
	}
}
